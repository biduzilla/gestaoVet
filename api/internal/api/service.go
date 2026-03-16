package api

import (
	"database/sql"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/internal/features/empresa"
)

type Services struct {
	empresa.EmpresaService
}

func NewServices(db *sql.DB, logger jsonlog.Logger) *Services {
	r := NewRepository(db, logger)
	return &Services{
		EmpresaService: empresa.NewService(r.Empresa, db),
	}
}
