package permissao

import (
	"database/sql"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/validator"
	"gestaoVet/utils"

	"github.com/google/uuid"
)

type cargoService struct {
	db         *sql.DB
	repository CargoRepository
}

type CargoService interface {
	FindByID(id uuid.UUID, cnpj string) (*Cargo, error)
	FindAll(nome string, f filters.Filters, cnpj string) ([]*Cargo, filters.Metadata, error)
	Insert(v *validator.Validator, model *Cargo, userID uuid.UUID) error
	Update(v *validator.Validator, model *Cargo, cnpj string, userID uuid.UUID) error
	Delete(id, userID uuid.UUID, cnpj string) error
}

func NewService(db *sql.DB,
	repository CargoRepository) CargoService {
	return &cargoService{
		repository: repository,
		db:         db,
	}
}

func (s *cargoService) FindByID(id uuid.UUID, cnpj string) (*Cargo, error) {
	return s.repository.FindByID(id, cnpj)
}

func (s *cargoService) FindAll(nome string, f filters.Filters, cnpj string) ([]*Cargo, filters.Metadata, error) {
	return s.repository.FindAll(nome, f, cnpj)
}

func (s *cargoService) Insert(v *validator.Validator, model *Cargo, userID uuid.UUID) error {
	return utils.RunInTx(s.db, func(tx *sql.Tx) error {
		if model.Validate(v); !v.Valid() {
			return e.ErrInvalidData
		}

		return s.repository.Insert(tx, model, userID)
	})
}

func (s *cargoService) Update(v *validator.Validator, model *Cargo, cnpj string, userID uuid.UUID) error {
	return utils.RunInTx(s.db, func(tx *sql.Tx) error {
		if model.Validate(v); !v.Valid() {
			return e.ErrInvalidData
		}

		return s.repository.Update(tx, model, cnpj, userID)
	})
}

func (s *cargoService) Delete(id, userID uuid.UUID, cnpj string) error {
	return utils.RunInTx(s.db, func(tx *sql.Tx) error {
		return s.repository.Delete(tx, id, userID, cnpj)
	})
}
