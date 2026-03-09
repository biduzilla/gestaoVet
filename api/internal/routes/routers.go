package routes

import (
	"database/sql"
	"gestaoVet/internal/config"
	"gestaoVet/internal/jsonlog"

	"github.com/go-chi/chi"
)

type Router struct {
}

func NewRouter(
	db *sql.DB,
	logger jsonlog.Logger,
	config config.Config,
) *Router {
	return &Router{}
}

func (router *Router) RegisterRoutes() *chi.Mux
