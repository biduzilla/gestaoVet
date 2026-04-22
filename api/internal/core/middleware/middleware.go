package middleware

import (
	"context"
	"expvar"
	"fmt"
	"gestaoVet/internal/core/config"
	"gestaoVet/internal/core/contexts"
	"gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/interfaces"
	"gestaoVet/internal/core/jsonlog"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var (
	totalRequestsReceived           = expvar.NewInt("total_requests_received")
	totalResponsesSent              = expvar.NewInt("total_responses_sent")
	totalProcessingTimeMicroseconds = expvar.NewInt("total_processing_time_μs")
	totalResponsesSentByStatus      = expvar.NewMap("total_responses_sent_by_status")
)

type JWTService interface {
	ExtractAuthenticatedUser(tokenString string) (interfaces.User, error)
}

type middleware struct {
	errHandler errors.ErrorHandler
	jwtService JWTService
	config     config.Config
	logger     jsonlog.Logger
	shutdown   <-chan struct{}
}

type Middleware interface {
	Metrics(next http.Handler) http.Handler
	EnableCORS(next http.Handler) http.Handler
	RequireAuthenticatedUser(next http.Handler) http.Handler
	RequireActivatedUser(next http.Handler) http.Handler
	Authenticate(next http.Handler) http.Handler
	RateLimit(next http.Handler) http.Handler
	RecoverPanic(next http.Handler) http.Handler
	Logging(next http.Handler) http.Handler
	RequirePermission(codes []interfaces.Role) func(http.Handler) http.Handler
	TimeoutMiddleWare(next http.Handler) http.Handler
}

func New(
	errHandler errors.ErrorHandler,
	config config.Config,
	jwtService JWTService,
	logger jsonlog.Logger,
	shutdown <-chan struct{},
) *middleware {
	return &middleware{
		jwtService: jwtService,
		errHandler: errHandler,
		config:     config,
		logger:     logger,
		shutdown:   shutdown,
	}
}

type metricsResponseWriter struct {
	wrapped       http.ResponseWriter
	statusCode    int
	headerWritten bool
}

func newMetricsResponseWriter(w http.ResponseWriter) *metricsResponseWriter {
	return &metricsResponseWriter{
		wrapped:    w,
		statusCode: http.StatusOK,
	}
}

func (mw *metricsResponseWriter) Header() http.Header {
	return mw.wrapped.Header()
}

func (mw *metricsResponseWriter) WriteHeader(statusCode int) {
	mw.wrapped.WriteHeader(statusCode)
	if !mw.headerWritten {
		mw.statusCode = statusCode
		mw.headerWritten = true
	}
}

func (mw *metricsResponseWriter) Write(b []byte) (int, error) {
	mw.headerWritten = true
	return mw.wrapped.Write(b)
}

func (mw *metricsResponseWriter) Unwrap() http.ResponseWriter {
	return mw.wrapped
}

func (m *middleware) Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		totalRequestsReceived.Add(1)

		mw := newMetricsResponseWriter(w)
		next.ServeHTTP(mw, r)

		totalResponsesSent.Add(1)
		totalResponsesSentByStatus.Add(strconv.Itoa(mw.statusCode), 1)

		duration := time.Since(start).Microseconds()
		totalProcessingTimeMicroseconds.Add(duration)
	})
}

func (m *middleware) EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Access-Control-Request-Method")
		origin := r.Header.Get("Origin")
		if origin != "" {
			for i := range m.config.CORS.TrustedOrigins {
				if origin == m.config.CORS.TrustedOrigins[i] {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
						w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
						w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
						w.WriteHeader(http.StatusOK)
						return
					}
					break
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (m *middleware) RequireAuthenticatedUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := contexts.ContextGetUser(r.Context())
		fmt.Printf("token: %s\n", user)

		if user.IsAnonymous() {
			m.errHandler.AuthenticationRequiredResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *middleware) RequireActivatedUser(next http.Handler) http.Handler {
	return m.RequireAuthenticatedUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := contexts.ContextGetUser(r.Context())
		if !user.GetIsAtivo() {
			m.errHandler.InactiveAccountResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func (m *middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")
		authorizationHeader := r.Header.Get("Authorization")

		if authorizationHeader == "" {
			r = contexts.ContextSetUser(r, interfaces.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			m.errHandler.InvalidCredentialsResponse(w, r)
			return
		}

		token := headerParts[1]
		user, err := m.jwtService.ExtractAuthenticatedUser(token)

		if err != nil {
			m.handleTokenError(w, r, err)
			return
		}

		if user == nil {
			m.errHandler.InvalidAuthenticationTokenResponse(w, r)
			return
		}

		if !user.GetIsAtivo() {
			m.errHandler.InactiveAccountResponse(w, r)
			return
		}

		r = contexts.ContextSetUser(r, user)

		next.ServeHTTP(w, r)
	})
}

func (m *middleware) handleTokenError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case strings.Contains(err.Error(), "malformed"):
		m.errHandler.MalFormedTokenResponse(w, r)
	case strings.Contains(err.Error(), "expired"):
		m.errHandler.ExpiredTokenResponse(w, r)
	default:
		m.errHandler.HandlerError(w, r, err)
	}
}

func (m *middleware) RequirePermission(codes []interfaces.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			user := contexts.ContextGetUser(r.Context())

			permissions := user.GetRoles()

			hasPermission := false

			for _, code := range codes {
				if slices.Contains(permissions, code) {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				m.errHandler.NotPermittedResponse(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (m *middleware) RateLimit(next http.Handler) http.Handler {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				mu.Lock()
				for ip, c := range clients {
					if time.Since(c.lastSeen) > 3*time.Minute {
						delete(clients, ip)
					}
				}
				mu.Unlock()
			case <-m.shutdown:
				mu.Lock()
				clients = make(map[string]*client) // Limpa tudo
				mu.Unlock()
				return
			}
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.config.Limiter.Enabled {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				m.errHandler.ServerErrorResponse(w, r, err)
				return
			}

			mu.Lock()

			if _, found := clients[ip]; !found {
				clients[ip] = &client{
					limiter: rate.NewLimiter(
						rate.Limit(m.config.Limiter.RPS),
						m.config.Limiter.Burst,
					),
				}
			}
			clients[ip].lastSeen = time.Now()
			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				m.errHandler.RateLimitExceededResponse(w, r)
				return
			}
			mu.Unlock()
		}
		next.ServeHTTP(w, r)
	})
}

func (m *middleware) RecoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				m.errHandler.ServerErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (m *middleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		mw := newMetricsResponseWriter(w)

		next.ServeHTTP(mw, r)

		m.logger.PrintInfo("request processed", map[string]string{
			"method":   r.Method,
			"path":     r.URL.Path,
			"remote":   r.RemoteAddr,
			"status":   http.StatusText(mw.statusCode),
			"duration": time.Since(start).String(),
		})
	})
}

func (m *middleware) TimeoutMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(m.config.Server.Timeout))
		defer cancel()

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}
