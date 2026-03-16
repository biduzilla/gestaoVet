package api

import (
	"database/sql"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/internal/features/empresa"
	"gestaoVet/internal/features/usuario"
)

type Repositories struct {
	Empresa empresa.EmpresaRepository
	Usuario usuario.UsuarioRepository
}

func NewRepository(
	db *sql.DB,
	logger jsonlog.Logger,
) *Repositories {
	return &Repositories{
		Empresa: empresa.NewRepository(db, logger),
		Usuario: usuario.NewRepository(db, logger),
	}
}
