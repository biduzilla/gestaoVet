package api

import (
	"database/sql"
	"gestaoVet/internal/core/config"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/internal/features/auth"
	"gestaoVet/internal/features/empresa"
	"gestaoVet/internal/features/tutor"
	"gestaoVet/internal/features/usuario"
)

type Services struct {
	empresa.EmpresaService
	usuario.UsuarioService
	auth.AuthService
	tutor.TutorService
}

func NewServices(db *sql.DB, logger jsonlog.Logger, config config.Config) (*Services, error) {
	r := NewRepository(db, logger)
	usuarioService := usuario.NewService(r.UsuarioRepository, db)
	authService, err := auth.NewService(usuarioService, config)
	if err != nil {
		return nil, err
	}

	return &Services{
		EmpresaService: empresa.NewService(usuarioService, r.EmpresaRepository, db),
		UsuarioService: usuarioService,
		AuthService:    authService,
		TutorService:   tutor.NewService(r.TutorRepository, db),
	}, nil
}
