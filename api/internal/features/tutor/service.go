package tutor

import (
	"context"
	"database/sql"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/transaction"
	"gestaoVet/internal/core/validator"

	"github.com/google/uuid"
)

type tutorService struct {
	repo TutorRepository
	tx   transaction.Manager
}

type TutorService interface {
	FindByID(
		ctx context.Context,
		ID uuid.UUID,
	) (*Tutor, error)

	FindAllBySearch(
		ctx context.Context,
		search string,
		f filters.Filters,
	) ([]*Tutor, filters.Metadata, error)

	Save(
		ctx context.Context,
		model *Tutor,
		tx *sql.Tx,
	) error

	Update(
		ctx context.Context,
		model *Tutor,
		tx *sql.Tx,
	) error

	DeleteByID(
		ctx context.Context,
		ID uuid.UUID,
		tx *sql.Tx,
	) error
}

func NewService(
	repo TutorRepository,
	tx transaction.Manager,
) TutorService {
	return &tutorService{
		repo: repo,
		tx:   tx,
	}
}

func (s *tutorService) FindByID(
	ctx context.Context,
	ID uuid.UUID,
) (*Tutor, error) {
	return s.repo.FindByID(ctx, ID)
}

func (s *tutorService) FindAllBySearch(
	ctx context.Context,
	search string,
	f filters.Filters,
) ([]*Tutor, filters.Metadata, error) {
	return s.repo.FindAll(ctx, search, f)
}

func (s *tutorService) Save(
	ctx context.Context,
	model *Tutor,
	tx *sql.Tx,
) error {
	saveLogic := func(tx *sql.Tx) error {
		v := validator.New()
		if model.Validate(v); !v.Valid() {
			return e.NewValidationError(v.Errors)
		}

		return s.repo.Insert(ctx, tx, model)
	}

	if tx != nil {
		return saveLogic(tx)
	}

	return s.tx.RunInTx(ctx, saveLogic)
}

func (s *tutorService) Update(
	ctx context.Context,
	model *Tutor,
	tx *sql.Tx,
) error {
	updateLogic := func(tx *sql.Tx) error {
		v := validator.New()

		if model.Validate(v); !v.Valid() {
			return e.NewValidationError(v.Errors)
		}
		return s.repo.Update(ctx, tx, model)
	}

	if tx != nil {
		return updateLogic(tx)
	}

	return s.tx.RunInTx(ctx, updateLogic)
}

func (s *tutorService) DeleteByID(
	ctx context.Context,
	ID uuid.UUID,
	tx *sql.Tx,
) error {
	deleteLogic := func(tx *sql.Tx) error {
		return s.repo.DeleteByID(ctx, ID, tx)
	}

	if tx != nil {
		return deleteLogic(tx)
	}

	return s.tx.RunInTx(ctx, deleteLogic)
}
