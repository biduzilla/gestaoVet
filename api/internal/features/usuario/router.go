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
		router.Post("/", r.handler.Save)

		router.Group(func(router chi.Router) {
			router.Use(r.m.RequireActivatedUser)

			router.Use()
			router.Get("/{id}", r.handler.FindByID)
			router.Get("/", r.handler.FindByAll)
			router.Post("/", r.handler.Save)

			router.With(
				r.m.RequirePermission([]int{int(ROLE_ADMIN)}),
			).Put("/", r.handler.Update)

			router.With(
				r.m.RequirePermission([]int{int(ROLE_ADMIN)}),
			).Delete("/{id}", r.handler.Delete)
		})

	})
}
