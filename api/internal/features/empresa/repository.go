package empresa

import (
	"context"
	"database/sql"
	"gestaoVet/internal/core/contexts"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/internal/core/repository"

	"github.com/lib/pq"
)

type empresaRepository struct {
	db             *sql.DB
	logger         jsonlog.Logger
	baseRepository repository.BaseRepository[Empresa]
}

func NewRepository(
	db *sql.DB,
	logger jsonlog.Logger,
) EmpresaRepository {
	return &empresaRepository{
		db:             db,
		logger:         logger,
		baseRepository: repository.NewBaseRepository[Empresa](db, logger, "empresas", "e"),
	}
}

type EmpresaRepository interface {
	FindByCnpj(
		ctx context.Context,
		cnpj string,
	) (*Empresa, error)

	FindAll(
		ctx context.Context,
		cnpj, nomeFantasia, razaoSocial, email string,
		f filters.Filters,
	) ([]*Empresa, filters.Metadata, error)

	Insert(
		ctx context.Context,
		tx *sql.Tx,
		model *Empresa,
	) error

	Update(
		ctx context.Context,
		tx *sql.Tx,
		model *Empresa,
	) error

	Delete(
		ctx context.Context,
		tx *sql.Tx,
	) error
}

func parseEmpresaConstraintError(err error) error {
	if pqErr, ok := err.(*pq.Error); ok {
		switch pqErr.Constraint {
		case "empresas_pkey":
			return e.ValidationAlreadyExists("cnpj")
		case "empresas_email_key":
			return e.ValidationAlreadyExists("email")
		case "empresas_razao_social_key":
			return e.ValidationAlreadyExists("razao_social")
		case "empresas_nome_fantasia_key":
			return e.ValidationAlreadyExists("nome_fantasia")
		case "empresas_nome_telefone_key":
			return e.ValidationAlreadyExists("telefone")
		}
	}
	return err
}

func (r *empresaRepository) FindByCnpj(
	ctx context.Context,
	cnpj string,
) (*Empresa, error) {
	query := `e.cnpj = :cnpj`
	params := map[string]any{
		"cnpj": cnpj,
	}
	return r.baseRepository.FindOne(ctx, repository.WithQueryExtraWhere(query, params))
}

func (r *empresaRepository) FindAll(
	ctx context.Context,
	cnpj, nomeFantasia, razaoSocial, email string,
	f filters.Filters,
) ([]*Empresa, filters.Metadata, error) {
	query := `(to_tsvector('simple', e.cnpj) @@ plainto_tsquery('simple', :cnpj) OR :cnpj = '')
            and (to_tsvector('simple', e.nome_fantasia) @@ plainto_tsquery('simple', :nomeFantasia) OR :nomeFantasia = '')
            and (to_tsvector('simple', e.razao_social) @@ plainto_tsquery('simple', :razaoSocial) OR :razaoSocial = '')
            and (to_tsvector('simple', e.email) @@ plainto_tsquery('simple', :email) OR :email = '')
			and e.deleted = false`

	params := map[string]any{
		"cnpj":         cnpj,
		"nomeFantasia": nomeFantasia,
		"razaoSocial":  razaoSocial,
		"email":        email,
	}

	return r.baseRepository.FindWithFilters(ctx, f, repository.WithQueryExtraWhere(query, params))
}

func (r *empresaRepository) Insert(
	ctx context.Context,
	tx *sql.Tx,
	model *Empresa,
) error {
	err := r.baseRepository.Insert(
		ctx,
		tx,
		model,
	)

	if err != nil {
		return parseEmpresaConstraintError(err)
	}
	return nil
}

func (r *empresaRepository) Update(
	ctx context.Context,
	tx *sql.Tx,
	model *Empresa,
) error {
	err := r.baseRepository.Update(
		ctx,
		tx,
		model,
		model.Cnpj,
	)

	if err != nil {
		return parseEmpresaConstraintError(err)
	}

	return nil
}

func (r *empresaRepository) Delete(
	ctx context.Context,
	tx *sql.Tx,
) error {
	user := contexts.ContextGetUser(ctx)

	query := `cnpj = :cnpj`
	params := map[string]any{
		"cnpj":   user.GetCNPJ(),
		"userID": user.GetID(),
	}

	return r.baseRepository.DeleteByQuery(ctx, tx, repository.WithQueryExtraWhere(query, params))
}
