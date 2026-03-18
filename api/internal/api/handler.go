package api

import (
	"database/sql"
	"gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/internal/features/empresa"
)

type handlers struct {
	Empresa empresa.EmpresaHandler
}

func NewHandler(
	db *sql.DB,
	logger jsonlog.Logger,
	errHandler errors.ErrorHandler,
) *handlers {
	s := NewServices(db, logger)
	return &handlers{
		Empresa: empresa.NewHandler(s.EmpresaService, errHandler),
	}
}
