package usuario

import (
	"gestaoVet/internal/core/middleware"

	"github.com/go-chi/chi"
)

type usuarioRouter struct {
	handler UsuarioHandler
	m       middleware.Middleware
}

type UsuarioRouter interface {
	Routes(router chi.Router)
}

func NewRouter(
	handler UsuarioHandler,
	m middleware.Middleware,
) *usuarioRouter {
	return &usuarioRouter{
		handler: handler,
		m:       m,
	}
}

func (r *usuarioRouter) Routes(router chi.Router) {
	router.Route("/usuario", func(router chi.Router) {
		router.Use(r.m.RequireActivatedUser)

		router.Get("/{id}", r.handler.FindByID)
		router.Get("/year", r.handler.FindByAll)
		router.Post("/", r.handler.Save)
		router.Put("/", r.handler.Update)
		router.Delete("/{id}", r.handler.Delete)
	})
}
