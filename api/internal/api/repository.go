package api

import (
	"database/sql"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/internal/features/empresa"
	"gestaoVet/internal/features/tutor"
	"gestaoVet/internal/features/usuario"
)

type Repositories struct {
	empresa.EmpresaRepository
	usuario.UsuarioRepository
	tutor.TutorRepository
}

func NewRepository(
	db *sql.DB,
	logger jsonlog.Logger,
) *Repositories {
	return &Repositories{
		EmpresaRepository: empresa.NewRepository(db, logger),
		UsuarioRepository: usuario.NewRepository(db, logger),
		TutorRepository:   tutor.NewRepository(db, logger),
	}
}
