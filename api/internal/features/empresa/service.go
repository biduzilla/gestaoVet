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
	Save(model *Empresa, v *validator.Validator, userID uuid.UUID) error
	FindByID(id uuid.UUID) (*Empresa, error)
	Delete(id, userID uuid.UUID) error
}

func (s *empresaService) FindAll(
	cnpj, nomeFantasia, razaoSocial, email string,
	f filters.Filters,
) ([]*Empresa, filters.Metadata, error) {
	return s.repository.FindAll(cnpj, nomeFantasia, razaoSocial, email, f)
}

func (s *empresaService) Save(model *Empresa, v *validator.Validator, userID uuid.UUID) error {
	return utils.RunInTx(s.db, func(tx *sql.Tx) error {
		if model.Validate(v); !v.Valid() {
			return errors.ErrInvalidData
		}

		return s.repository.InsertOrUpdate(tx, model, userID)
	})
}

func (s *empresaService) FindByID(id uuid.UUID) (*Empresa, error) {
	return s.repository.FindByID(id)
}

func (s *empresaService) Delete(id, userID uuid.UUID) error {
	return utils.RunInTx(s.db, func(tx *sql.Tx) error {
		return s.repository.Delete(tx, id, userID)
	})
}
