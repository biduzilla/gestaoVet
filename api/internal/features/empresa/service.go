package empresa

import (
	"database/sql"
	"gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/validator"
	"gestaoVet/utils"

	"github.com/google/uuid"
)

type empresaService struct {
	repository EmpresaRepository
	db         *sql.DB
}

func NewService(
	repository EmpresaRepository,
	db *sql.DB,
) *empresaService {
	return &empresaService{
		repository: repository,
		db:         db,
	}
}

type EmpresaService interface {
	FindAll(
		cnpj, nomeFantasia, razaoSocial, email string,
		f filters.Filters,
	) ([]*Empresa, filters.Metadata, error)
	FindByCnpj(cnpj string) (*Empresa, error)
	Save(model *Empresa, v *validator.Validator) error
	Update(model *Empresa, v *validator.Validator, userID uuid.UUID, cnpj string) error
	Delete(cnpj string, userID uuid.UUID) error
}

func (s *empresaService) FindAll(
	cnpj, nomeFantasia, razaoSocial, email string,
	f filters.Filters,
) ([]*Empresa, filters.Metadata, error) {
	return s.repository.FindAll(cnpj, nomeFantasia, razaoSocial, email, f)
}

func (s *empresaService) Save(model *Empresa, v *validator.Validator) error {
	return utils.RunInTx(s.db, func(tx *sql.Tx) error {
		if model.Validate(v); !v.Valid() {
			return errors.ErrInvalidData
		}

		return s.repository.Insert(tx, model)
	})
}

func (s *empresaService) Update(model *Empresa, v *validator.Validator, userID uuid.UUID, cnpj string) error {
	return utils.RunInTx(s.db, func(tx *sql.Tx) error {
		if model.Validate(v); !v.Valid() {
			return errors.ErrInvalidData
		}

		return s.repository.Update(tx, model, userID, cnpj)
	})
}

func (s *empresaService) FindByCnpj(cnpj string) (*Empresa, error) {
	return s.repository.FindByCnpj(cnpj)
}

func (s *empresaService) Delete(cnpj string, userID uuid.UUID) error {
	return utils.RunInTx(s.db, func(tx *sql.Tx) error {
		return s.repository.Delete(tx, cnpj, userID)
	})
}
