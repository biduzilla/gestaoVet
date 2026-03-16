package api

import (
	"database/sql"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/internal/features/empresa"
)

type Repositories struct {
	Empresa empresa.EmpresaRepository
}

func NewRepository(
	db *sql.DB,
	logger jsonlog.Logger,
) *Repositories {
	return &Repositories{
		Empresa: empresa.NewRepository(db, logger),
	}
}
