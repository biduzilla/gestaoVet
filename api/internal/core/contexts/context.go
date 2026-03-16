package contexts

import (
	"context"
	"gestaoVet/internal/features/usuario"
	"net/http"
)

type contextKey string

const userContextKey = contextKey("user")

func ContextSetUser(r *http.Request, user *usuario.Usuario) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

func ContextGetUser(r *http.Request) *usuario.Usuario {
	user, ok := r.Context().Value(userContextKey).(*usuario.Usuario)
	if !ok {
		panic("missing user value in request context")
	}
	return user
}
