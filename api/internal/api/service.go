package api

import (
	"database/sql"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/internal/features/empresa"
	"gestaoVet/internal/features/usuario"
)

type Services struct {
	empresa.EmpresaService
	usuario.UsuarioService
}

func NewServices(db *sql.DB, logger jsonlog.Logger) *Services {
	r := NewRepository(db, logger)
	return &Services{
		EmpresaService: empresa.NewService(r.Empresa, db),
		UsuarioService: usuario.NewService(r.Usuario, db),
	}
}
