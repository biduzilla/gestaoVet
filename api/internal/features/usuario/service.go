package usuario

import (
	"database/sql"
	"gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/validator"
	"gestaoVet/utils"

	"github.com/google/uuid"
)

type usuarioService struct {
	repository UsuarioRepository
	db         *sql.DB
}

type UsuarioService interface {
	FindByID(
		ID uuid.UUID,
		cnpj string,
	) (*Usuario, error)

	FindAll(
		nome, telefone, email, cnpj string,
		f filters.Filters,
	) ([]*Usuario, filters.Metadata, error)

	Save(
		v *validator.Validator,
		model *Usuario,
		cnpj string,
		ID *uuid.UUID,
	) error

	Delete(tx *sql.Tx,
		id,
		userID uuid.UUID,
		cnpj string,
	) error
}

func NewService(
	repository UsuarioRepository,
	db *sql.DB,
) *usuarioService {
	return &usuarioService{
		repository: repository,
		db:         db,
	}
}

func (s *usuarioService) FindByID(
	ID uuid.UUID,
	cnpj string,
) (*Usuario, error) {
	return s.repository.FindByID(ID, cnpj)
}

func (s *usuarioService) FindAll(
	nome, telefone, email, cnpj string,
	f filters.Filters,
) ([]*Usuario, filters.Metadata, error) {
	return s.repository.FindAll(nome, telefone, email, cnpj, f)
}

func (s *usuarioService) Save(
	v *validator.Validator,
	model *Usuario,
	cnpj string,
	ID *uuid.UUID,
) error {
	return utils.RunInTx(s.db, func(tx *sql.Tx) error {
		if model.Validate(v); !v.Valid() {
			return errors.ErrInvalidData
		}

		return s.repository.InsertOrUpdate(tx, model, cnpj, ID)
	})
}

func (s *usuarioService) Delete(tx *sql.Tx,
	id,
	userID uuid.UUID,
	cnpj string,
) error {
	return utils.RunInTx(s.db, func(tx *sql.Tx) error {
		return s.repository.Delete(tx, id, userID, cnpj)
	})
}
