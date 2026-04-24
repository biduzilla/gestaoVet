package pet

import (
	"context"
	"database/sql"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/transaction"
	"gestaoVet/internal/core/validator"

	"github.com/google/uuid"
)

type petService struct {
	repo PetReposistory
	tx   transaction.Manager
}

type PetService interface {
	FindByID(ctx context.Context, ID uuid.UUID) (*Pet, error)
	FindAll(ctx context.Context, f filters.Filters, search ...string) ([]*Pet, filters.Metadata, error)
	Save(
		ctx context.Context,
		model *Pet,
		tx *sql.Tx,
	) error
	Update(
		ctx context.Context,
		model *Pet,
		tx *sql.Tx,
	) error
	DeleteBy(
		ctx context.Context, id uuid.UUID,
	) error
}

func NewService(
	repo PetReposistory,
	tx transaction.Manager,
) PetService {
	return &petService{
		repo: repo,
		tx:   tx,
	}
}

func (s *petService) FindByID(ctx context.Context, ID uuid.UUID) (*Pet, error) {
	return s.repo.FindByID(ctx, ID)
}

func (s *petService) FindAll(ctx context.Context, f filters.Filters, search ...string) ([]*Pet, filters.Metadata, error) {
	return s.repo.FindAll(ctx, f, search...)
}

func (s *petService) Save(
	ctx context.Context,
	model *Pet,
	tx *sql.Tx,
) error {
	fn := func(tx *sql.Tx) error {
		v := validator.New()
		if model.Validate(v); !v.Valid() {
			return e.NewValidationError(v.Errors)
		}
		return s.repo.Insert(ctx, tx, model)
	}

	if tx != nil {
		return fn(tx)
	}

	return s.tx.RunInTx(ctx, fn)
}

func (s *petService) Update(
	ctx context.Context,
	model *Pet,
	tx *sql.Tx,
) error {
	fn := func(tx *sql.Tx) error {
		v := validator.New()
		if model.Validate(v); !v.Valid() {
			return e.NewValidationError(v.Errors)
		}
		return s.repo.Update(ctx, tx, model)
	}

	if tx != nil {
		return fn(tx)
	}

	return s.tx.RunInTx(ctx, fn)
}

func (s *petService) DeleteBy(
	ctx context.Context, id uuid.UUID,
) error {
	return s.tx.RunInTx(ctx, func(tx *sql.Tx) error {
		return s.repo.DeleteByID(ctx, id, tx)
	})
}
