package api

import (
	"database/sql"
	"expvar"
	"gestaoVet/internal/core/adapter"
	"gestaoVet/internal/core/config"
	"gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/internal/core/middleware"
	"gestaoVet/internal/features/auth"
	"gestaoVet/internal/features/empresa"
	"net/http"

	"github.com/go-chi/chi"
)

type Router struct {
	errHandler errors.ErrorHandler
	m          middleware.Middleware
	empresa    empresa.EmpresaRouter
	auth       auth.AuthRouter
}

func NewRouter(
	db *sql.DB,
	logger jsonlog.Logger,
	config config.Config,
) *Router {
	e := errors.NewErrorHandler(logger)
	h := NewHandler(db, logger, e, config)
	m := middleware.New(
		e,
		config,
		adapter.UserFinderAdapter{
			Service: h.Services.UsuarioService,
		},
		h.Services.AuthService,
	)
	return &Router{
		m:          m,
		errHandler: e,
		empresa:    empresa.NewRouter(h.Empresa, m),
		auth:       auth.NewRouter(h.Auth, m),
	}
}

func (router *Router) RegisterRoutes() *chi.Mux {
	r := chi.NewRouter()
	r.Use(router.m.RecoverPanic)
	r.Use(router.m.Metrics)
	r.Use(router.m.RateLimit)
	r.Use(router.m.EnableCORS)
	r.Use(router.m.Authenticate)

	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		router.errHandler.NotFoundResponse(w, req)
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, req *http.Request) {
		router.errHandler.MethodNotAllowedResponse(w, req)
	})

	r.Route("/v1", func(r chi.Router) {
		r.Mount("/debug/vars", expvar.Handler())
		router.empresa.Routes(r)
	})

	return r
}
