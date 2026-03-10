package services

import (
	"database/sql"
	"gestaoVet/internal/models"
	"gestaoVet/internal/models/filters"
	"gestaoVet/internal/repositories"
	"gestaoVet/utils"
	"gestaoVet/utils/errors"
	"gestaoVet/utils/validator"

	"github.com/google/uuid"
)

type empresaService struct {
	repository repositories.EmpresaRepository
	db         *sql.DB
}

func NewEmpresaService(
	repository repositories.EmpresaRepository,
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
	) ([]*models.Empresa, filters.Metadata, error)
	Save(model *models.Empresa, v *validator.Validator) error
	FindByID(id uuid.UUID) (*models.Empresa, error)
	Delete(id, userID uuid.UUID) error
}

func (s *empresaService) FindAll(
	cnpj, nomeFantasia, razaoSocial, email string,
	f filters.Filters,
) ([]*models.Empresa, filters.Metadata, error) {
	return s.repository.FindAll(cnpj, nomeFantasia, razaoSocial, email, f)
}

func (s *empresaService) Save(model *models.Empresa, v *validator.Validator) error {
	return utils.RunInTx(s.db, func(tx *sql.Tx) error {
		if model.Validate(v); !v.Valid() {
			return errors.ErrInvalidData
		}

		return s.repository.InsertOrUpdate(tx, model)
	})
}
func (s *empresaService) FindByID(id uuid.UUID) (*models.Empresa, error) {
	return s.repository.FindByID(id)
}
func (s *empresaService) Delete(id, userID uuid.UUID) error {
	return utils.RunInTx(s.db, func(tx *sql.Tx) error {
		return s.repository.Delete(tx, id, userID)
	})
}
