package services

import (
	"database/sql"
	"gestaoVet/internal/jsonlog"
	"gestaoVet/internal/repositories"
)

type Services struct {
	EmpresaService
}

func NewServices(db *sql.DB, logger jsonlog.Logger) *Services {
	r := repositories.New(db, logger)
	return &Services{
		EmpresaService: NewEmpresaService(r.Empresa, db),
	}
}
