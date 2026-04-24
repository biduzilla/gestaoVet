package tutor

import (
	"context"
	"database/sql"
	"gestaoVet/internal/core/contexts"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/internal/core/repository"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type tutorRepository struct {
	db             *sql.DB
	logger         jsonlog.Logger
	baseRepository repository.BaseRepository[Tutor]
}
type TutorRepository interface {
	FindByID(
		ctx context.Context,
		ID uuid.UUID,
	) (*Tutor, error)

	FindAll(
		ctx context.Context,
		search string,
		f filters.Filters,
	) ([]*Tutor, filters.Metadata, error)

	Insert(
		ctx context.Context,
		tx *sql.Tx,
		model *Tutor,
	) error

	Update(
		ctx context.Context,
		tx *sql.Tx,
		model *Tutor,
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
) TutorRepository {
	return &tutorRepository{
		db:             db,
		logger:         logger,
		baseRepository: repository.NewBaseRepository[Tutor](db, logger, "tutores", "t"),
	}
}

func parseTutorConstraintError(err error) error {
	if pqErr, ok := err.(*pq.Error); ok {
		switch pqErr.Constraint {
		case "tutor_identidade_key":
			return e.ValidationAlreadyExists("identidade")
		case "tutor_cpf_key":
			return e.ValidationAlreadyExists("cpf")
		case "tutor_celular_key":
			return e.ValidationAlreadyExists("celular")
		}
	}
	return err
}

func (r *tutorRepository) FindByID(
	ctx context.Context,
	ID uuid.UUID,
) (*Tutor, error) {
	user := contexts.ContextGetUser(ctx)
	where := `
		t.id = :id 
		and t.cnpj = :cnpj
	`
	params := map[string]any{
		"id":   ID,
		"cnpj": user.GetCNPJ(),
	}

	return r.baseRepository.FindOne(
		ctx,
		repository.WithQueryExtraWhere(where, params),
	)
}

func (r *tutorRepository) FindAll(
	ctx context.Context,
	search string,
	f filters.Filters,
) ([]*Tutor, filters.Metadata, error) {
	user := contexts.ContextGetUser(ctx)
	params := map[string]any{
		"search": search,
		"cnpj":   user.GetCNPJ(),
	}

	query := `
			(
				(:search is null or :search = '')
				or to_tsvector('simple', t.nome) @@ plainto_tsquery('simple', :search)
				or to_tsvector('simple', t.cpf) @@ plainto_tsquery('simple', :search)
				or to_tsvector('simple', t.telefone1) @@ plainto_tsquery('simple', :search) 
				or to_tsvector('simple', t.telefone2) @@ plainto_tsquery('simple', :search) 
				or to_tsvector('simple', t.email1) @@ plainto_tsquery('simple', :search) 
				or to_tsvector('simple', t.email2) @@ plainto_tsquery('simple', :search) 
				or to_tsvector('simple', t.identidade) @@ plainto_tsquery('simple', :search)
				)
			and t.cnpj = :cnpj
	`

	return r.baseRepository.FindWithFilters(ctx, f, repository.WithQueryExtraWhere(query, params))
}

func (r *tutorRepository) Insert(
	ctx context.Context,
	tx *sql.Tx,
	model *Tutor,
) error {
	user := contexts.ContextGetUser(ctx)

	err := r.baseRepository.Insert(
		ctx,
		tx,
		model,
		repository.WithExtraWhere("", map[string]any{
			"cnpj":      user.GetCNPJ(),
			"createdBy": user.GetID(),
		}),
	)

	if err != nil {
		return parseTutorConstraintError(err)
	}
	return nil
}

func (r *tutorRepository) Update(
	ctx context.Context,
	tx *sql.Tx,
	model *Tutor,
) error {
	user := contexts.ContextGetUser(ctx)

	err := r.baseRepository.Update(
		ctx,
		tx,
		model,
		model.ID,
		repository.WithExtraWhere("and t.cnpj = :cnpj", map[string]any{
			"cnpj":      user.GetCNPJ(),
			"updatedBy": user.GetID(),
		}),
	)

	if err != nil {
		return parseTutorConstraintError(err)
	}
	return nil
}

func (r *tutorRepository) DeleteByID(
	ctx context.Context,
	ID uuid.UUID,
	tx *sql.Tx,
) error {
	user := contexts.ContextGetUser(ctx)
	params := map[string]any{
		"id":   ID,
		"cnpj": user.GetCNPJ(),
	}

	query := `
		t.id = :id
		and t.cnpj = :cnpj
	`
	return r.baseRepository.DeleteByQuery(ctx, tx, repository.WithQueryExtraWhere(query, params))
}
