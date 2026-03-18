package api

import (
	"database/sql"
	"gestaoVet/internal/core/config"
	"gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/internal/features/empresa"
)

type handlers struct {
	Services *Services
	Empresa  empresa.EmpresaHandler
}

func NewHandler(
	db *sql.DB,
	logger jsonlog.Logger,
	errHandler errors.ErrorHandler,
	config config.Config,
) *handlers {
	s := NewServices(db, logger, config)
	return &handlers{
		Services: s,
		Empresa:  empresa.NewHandler(s.EmpresaService, errHandler),
	}
}
