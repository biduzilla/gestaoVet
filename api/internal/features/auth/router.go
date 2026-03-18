package auth

import (
	"gestaoVet/internal/core/middleware"

	"github.com/go-chi/chi"
)

type authRouter struct {
	handler AuthHandler
	m       middleware.Middleware
}

type AuthRouter interface {
	Routes(router chi.Router)
}

func NewRouter(
	handler AuthHandler,
	m middleware.Middleware,
) *authRouter {
	return &authRouter{
		handler: handler,
		m:       m,
	}
}

func (r *authRouter) Routes(router chi.Router) {
	router.Route("/auth", func(router chi.Router) {
		router.Get("/", r.handler.Login)
		router.Get("/refresh-token", r.handler.RefreshToken)
	})
}
