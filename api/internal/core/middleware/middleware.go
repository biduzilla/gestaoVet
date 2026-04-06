package middleware

import (
	"expvar"
	"fmt"
	"gestaoVet/internal/core/config"
	"gestaoVet/internal/core/contexts"
	"gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/interfaces"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/internal/core/validator"
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

type UserFinder interface {
	FindByEmail(email string, v *validator.Validator) (interfaces.User, error)
}

type JWTService interface {
	ExtractUsername(tokenString string) (string, error)
}

type middleware struct {
	errHandler errors.ErrorHandler
	userFinder UserFinder
	jwtService JWTService
	config     config.Config
	logger     jsonlog.Logger
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
}

func New(
	errHandler errors.ErrorHandler,
	config config.Config,
	userFinder UserFinder,
	jwtService JWTService,
	logger jsonlog.Logger,
) *middleware {
	return &middleware{
		userFinder: userFinder,
		jwtService: jwtService,
		errHandler: errHandler,
		config:     config,
		logger:     logger,
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
		user := contexts.ContextGetUser(r)
		if user.IsAnonymous() {
			m.errHandler.AuthenticationRequiredResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *middleware) RequireActivatedUser(next http.Handler) http.Handler {
	return m.RequireAuthenticatedUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := contexts.ContextGetUser(r)
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
		username, err := m.jwtService.ExtractUsername(token)
		if err != nil {
			m.errHandler.InvalidAuthenticationTokenResponse(w, r)
			return
		}

		v := validator.New()
		user, err := m.userFinder.FindByEmail(username, v)
		if err != nil {
			m.errHandler.HandlerError(w, r, err, v)
			return
		}

		r = contexts.ContextSetUser(r, user)
		next.ServeHTTP(w, r)
	})
}

func (m *middleware) RequirePermission(codes []interfaces.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			user := contexts.ContextGetUser(r)

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
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
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
