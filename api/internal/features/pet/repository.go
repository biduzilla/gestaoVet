package pet

import (
	"context"
	"database/sql"
	"fmt"
	"gestaoVet/internal/core/contexts"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/internal/core/repository"
	"gestaoVet/internal/features/tutor"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type petRepository struct {
	db             *sql.DB
	logger         jsonlog.Logger
	baseRepository repository.BaseRepository[Pet]
}

type PetReposistory interface {
	FindByID(ctx context.Context, ID uuid.UUID) (*Pet, error)

	FindAll(
		ctx context.Context,
		f filters.Filters,
		search ...string,
	) ([]*Pet, filters.Metadata, error)

	Insert(
		ctx context.Context,
		tx *sql.Tx,
		model *Pet,
	) error

	Update(
		ctx context.Context,
		tx *sql.Tx,
		model *Pet,
	) error

	DeleteByID(
		ctx context.Context,
		ID uuid.UUID,
		tx *sql.Tx,
	) error
}

func NewRepository(
	db *sql.DB,
	logger jsonlog.Logger,
) PetReposistory {
	return &petRepository{
		db:             db,
		logger:         logger,
		baseRepository: repository.NewBaseRepository[Pet](db, logger, "pets", "p"),
	}
}

func (r *petRepository) parseConstraintError(err error) error {
	if pqErr, ok := err.(*pq.Error); ok {
		switch pqErr.Constraint {
		case "pet_microchip_key":
			return e.ValidationAlreadyExists("microchip")
		}
	}
	return err
}

func (r *petRepository) FindByID(ctx context.Context, ID uuid.UUID) (*Pet, error) {
	user := contexts.ContextGetUser(ctx)

	return r.baseRepository.FindById(
		ctx,
		ID,
		repository.WithQueryExtraWhere("p.cnpj = :cnpj", map[string]any{
			"cnpj": user.GetCNPJ(),
		}),
		repository.WithJoin(tutor.Tutor{}, "tutores", "t", "t.id = p.tutor_id"),
	)
}

func (r *petRepository) FindAll(
	ctx context.Context,
	f filters.Filters,
	search ...string,
) ([]*Pet, filters.Metadata, error) {
	user := contexts.ContextGetUser(ctx)
	query := fmt.Sprintf("%s and p.deleted = false", repository.BuildFilterQuery("p", search...))

	return r.baseRepository.FindWithFilters(
		ctx,
		f,
		repository.WithQueryExtraWhere(query, map[string]any{
			"cnpj": user.GetCNPJ(),
		}),
		repository.WithJoin(tutor.Tutor{}, "tutores", "t", "t.id = p.tutor_id"),
	)
}

func (r *petRepository) Insert(
	ctx context.Context,
	tx *sql.Tx,
	model *Pet,
) error {
	err := r.baseRepository.Insert(
		ctx,
		tx,
		model,
		repository.WithExtraFields([]string{"tutor_id"}, map[string]any{
			"tutor_id": model.Tutor.ID,
		},
		),
	)

	if err != nil {
		return r.parseConstraintError(err)
	}

	return nil
}

func (r *petRepository) Update(
	ctx context.Context,
	tx *sql.Tx,
	model *Pet,
) error {
	user := contexts.ContextGetUser(ctx)
	err := r.baseRepository.Update(
		ctx,
		tx,
		model,
		repository.WithExtraFields([]string{"tutor_id"}, map[string]any{
			"tutor_id": model.Tutor.ID,
		}),
		repository.WithExtraWhere("cnpj = :cnpj", map[string]any{
			"cnpj": user.GetCNPJ(),
		}),
	)

	if err != nil {
		return r.parseConstraintError(err)
	}

	return nil
}

func (r *petRepository) DeleteByID(
	ctx context.Context,
	ID uuid.UUID,
	tx *sql.Tx,
) error {
	user := contexts.ContextGetUser(ctx)
	return r.baseRepository.DeleteByQuery(
		ctx,
		tx,
		repository.WithQueryExtraWhere("id = :id and cnpj = :cnpj", map[string]any{
			"id":   ID,
			"cnpj": user.GetCNPJ(),
		}),
	)
}
