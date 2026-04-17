package tutor

import (
	"database/sql"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/validator"
	"gestaoVet/utils"

	"github.com/google/uuid"
)

type tutorService struct {
	repo TutorRepository
	db   *sql.DB
}

type TutorService interface {
	FindByID(
		ID uuid.UUID,
		cnpj string,
	) (*Tutor, error)

	FindAllBySearch(
		search string,
		cnpj string,
		f filters.Filters,
	) ([]*Tutor, filters.Metadata, error)

	Save(
		tx *sql.Tx, model *Tutor,
		v *validator.Validator,
		cpnj string,
	) error

	Update(
		tx *sql.Tx, model *Tutor,
		v *validator.Validator,
		cpnj string,
		ID uuid.UUID,
	) error

	DeleteByID(
		tx *sql.Tx,
		ID, userID uuid.UUID,
		cnpj string,
	) error
}

func NewService(
	repo TutorRepository,
	db *sql.DB,
) TutorService {
	return &tutorService{
		repo: repo,
		db:   db,
	}
}

func (s *tutorService) FindByID(
	ID uuid.UUID,
	cnpj string,
) (*Tutor, error) {
	return s.repo.FindByID(ID, cnpj)
}

func (s *tutorService) FindAllBySearch(
	search string,
	cnpj string,
	f filters.Filters,
) ([]*Tutor, filters.Metadata, error) {
	return s.repo.FindAll(search, cnpj, f)
}

func (s *tutorService) Save(
	tx *sql.Tx, model *Tutor,
	v *validator.Validator,
	cpnj string,
) error {
	saveLogic := func(tx *sql.Tx) error {
		model.Cnpj = cpnj
		if model.Validate(v); !v.Valid() {
			return e.ErrInvalidData
		}
		return s.repo.Insert(tx, model)
	}

	if tx != nil {
		return saveLogic(tx)
	}

	return utils.RunInTx(s.db, saveLogic)
}

func (s *tutorService) Update(
	tx *sql.Tx,
	model *Tutor,
	v *validator.Validator,
	cpnj string,
	ID uuid.UUID,
) error {
	updateLogic := func(tx *sql.Tx) error {
		if model.Validate(v); !v.Valid() {
			return e.ErrInvalidData
		}
		return s.repo.Update(tx, model, cpnj, ID)
	}

	if tx != nil {
		return updateLogic(tx)
	}

	return utils.RunInTx(s.db, updateLogic)
}

func (s *tutorService) DeleteByID(
	tx *sql.Tx,
	ID, userID uuid.UUID,
	cnpj string,
) error {
	deleteLogic := func(tx *sql.Tx) error {
		return s.DeleteByID(tx, ID, userID, cnpj)
	}

	if tx != nil {
		return deleteLogic(tx)
	}

	return utils.RunInTx(s.db, deleteLogic)
}
