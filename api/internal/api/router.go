package api

import (
	"database/sql"
	"expvar"
	"gestaoVet/internal/core/config"
	"gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/internal/core/middleware"
	"gestaoVet/internal/features/auth"
	"gestaoVet/internal/features/empresa"
	"gestaoVet/internal/features/tutor"
	"gestaoVet/internal/features/usuario"
	"net/http"

	"github.com/go-chi/chi"
)

type Router struct {
	errHandler errors.ErrorHandler
	m          middleware.Middleware
	empresa    empresa.EmpresaRouter
	auth       auth.AuthRouter
	usuario    usuario.UsuarioRouter
	tutor      tutor.TutorRouter
}

func NewRouter(
	db *sql.DB,
	logger jsonlog.Logger,
	config config.Config,
	shutdown <-chan struct{},
) (*Router, error) {
	e := errors.NewErrorHandler(logger)
	h, err := NewHandler(db, logger, e, config)
	if err != nil {
		return nil, err
	}

	m := middleware.New(
		e,
		config,
		h.Services.AuthService,
		logger,
		shutdown,
	)
	return &Router{
		m:          m,
		errHandler: e,
		empresa:    empresa.NewRouter(h.Empresa, m),
		auth:       auth.NewRouter(h.Auth, m),
		tutor:      tutor.NewRouter(h.Tutor, m),
	}, nil
}

func (router *Router) RegisterRoutes() *chi.Mux {
	r := chi.NewRouter()
	r.Use(router.m.RecoverPanic)
	r.Use(router.m.TimeoutMiddleWare)
	r.Use(router.m.Metrics)
	r.Use(router.m.RateLimit)
	r.Use(router.m.EnableCORS)
	r.Use(router.m.Authenticate)
	r.Use(router.m.Logging)

	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		router.errHandler.NotFoundResponse(w, req)
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, req *http.Request) {
		router.errHandler.MethodNotAllowedResponse(w, req)
	})

	r.Route("/v1", func(r chi.Router) {
		r.Mount("/debug/vars", expvar.Handler())
		router.empresa.Routes(r)
		router.auth.Routes(r)
		router.usuario.Routes(r)
		router.tutor.Routes(r)
	})

	return r
}
