package tutor

import (
	"gestaoVet/internal/core/middleware"

	"github.com/go-chi/chi"
)

type tutorRouter struct {
	handler TutorHandler
	m       middleware.Middleware
}

type TutorRouter interface {
	Routes(router chi.Router)
}

func NewRouter(
	handler TutorHandler,
	m middleware.Middleware,
) TutorRouter {
	return &tutorRouter{
		handler: handler,
		m:       m,
	}
}

func (r *tutorRouter) Routes(router chi.Router) {
	router.Route("/tutor", func(router chi.Router) {
		router.Group(func(router chi.Router) {
			router.Use(r.m.RequireActivatedUser)

			router.Get("/{id}", r.handler.FindByID)
			router.Get("/", r.handler.FindByAll)
			router.Post("/", r.handler.Save)
			router.Put("/", r.handler.Update)
			router.Delete("/{id}", r.handler.DeleteByID)
		})
	})
}
