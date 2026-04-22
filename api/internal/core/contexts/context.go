package contexts

import (
	"context"
	"gestaoVet/internal/core/interfaces"
	"net/http"
)

type contextKey string

const userContextKey = contextKey("user")

func ContextSetUser(r *http.Request, user interfaces.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

func ContextGetUser(ctx context.Context) interfaces.User {
	user, ok := ctx.Value(userContextKey).(interfaces.User)
	if !ok {
		panic("missing user value in request context")
	}
	return user
}
