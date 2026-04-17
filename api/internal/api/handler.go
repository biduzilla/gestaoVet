package api

import (
	"database/sql"
	"gestaoVet/internal/core/config"
	"gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/internal/features/auth"
	"gestaoVet/internal/features/empresa"
	"gestaoVet/internal/features/tutor"
	"gestaoVet/internal/features/usuario"
)

type handlers struct {
	Services *Services
	Empresa  empresa.EmpresaHandler
	Auth     auth.AuthHandler
	Usuario  usuario.UsuarioHandler
	Tutor    tutor.TutorHandler
}

func NewHandler(
	db *sql.DB,
	logger jsonlog.Logger,
	errHandler errors.ErrorHandler,
	config config.Config,
) (*handlers, error) {
	s, err := NewServices(db, logger, config)
	if err != nil {
		return nil, err
	}
	return &handlers{
		Services: s,
		Empresa:  empresa.NewHandler(s.EmpresaService, errHandler),
		Auth:     auth.NewHandler(s.AuthService, errHandler),
		Usuario:  usuario.NewHandler(s.UsuarioService, errHandler),
		Tutor:    tutor.NewHandler(s.TutorService, errHandler),
	}, nil
}
