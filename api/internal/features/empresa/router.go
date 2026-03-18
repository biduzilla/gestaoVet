package empresa

import (
	"gestaoVet/internal/core/middleware"

	"github.com/go-chi/chi"
)

type empresaRouter struct {
	handler EmpresaHandler
	m       middleware.Middleware
}

type EmpresaRouter interface {
	Routes(router chi.Router)
}

func NewRouter(
	handler EmpresaHandler,
	m middleware.Middleware,
) *empresaRouter {
	return &empresaRouter{
		handler: handler,
		m:       m,
	}
}

func (r *empresaRouter) Routes(router chi.Router) {
	router.Route("/empresa", func(router chi.Router) {
		router.Use(r.m.RequireActivatedUser)

		router.Get("/{id}", r.handler.FindByID)
		router.Get("/year", r.handler.FindByAll)
		router.Post("/", r.handler.Save)
		router.Put("/", r.handler.Update)
		router.Delete("/{id}", r.handler.Delete)
	})
}
