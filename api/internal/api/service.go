package api

import (
	"database/sql"
	"gestaoVet/internal/core/config"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/internal/features/auth"
	"gestaoVet/internal/features/empresa"
	"gestaoVet/internal/features/usuario"
)

type Services struct {
	empresa.EmpresaService
	usuario.UsuarioService
	auth.AuthService
}

func NewServices(db *sql.DB, logger jsonlog.Logger, config config.Config) *Services {
	r := NewRepository(db, logger)
	usuarioService := usuario.NewService(r.Usuario, db)
	return &Services{
		EmpresaService: empresa.NewService(r.Empresa, db),
		UsuarioService: usuarioService,
		AuthService:    auth.NewService(usuarioService, config),
	}
}
