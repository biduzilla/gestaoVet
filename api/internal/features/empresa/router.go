package empresa

import (
	"gestaoVet/internal/core/interfaces"
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
		router.Post("/", r.handler.Save)

		router.Group(func(router chi.Router) {
			router.Use(r.m.RequireActivatedUser)

			router.Get("/{cnpj}", r.handler.FindByCnpj)
			router.Get("/", r.handler.FindByAll)

			router.Group(func(router chi.Router) {
				router.Use(r.m.RequirePermission([]interfaces.Role{interfaces.ROLE_ADMIN}))

				router.Put("/", r.handler.Update)
				router.Delete("/{cnpj}", r.handler.Delete)
			})
		})
	})
}
