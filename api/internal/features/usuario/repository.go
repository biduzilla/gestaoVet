package usuario

import (
	"context"
	"database/sql"
	"fmt"
	"gestaoVet/internal/core/contexts"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/internal/core/repository"
	"gestaoVet/utils"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type usuarioRepository struct {
	db             *sql.DB
	logger         jsonlog.Logger
	baseRepository repository.BaseRepository[Usuario]
}

type UsuarioRepository interface {
	FindByEmail(
		ctx context.Context,
		email string,
	) (*Usuario, error)

	FindByID(
		ctx context.Context,
		ID uuid.UUID,
	) (*Usuario, error)

	FindAll(
		ctx context.Context,
		nome, telefone, email, cnpj string,
		f filters.Filters,
	) ([]*Usuario, filters.Metadata, error)

	Insert(
		ctx context.Context,
		tx *sql.Tx,
		model *Usuario,
	) error

	Update(
		ctx context.Context,
		tx *sql.Tx,
		model *Usuario,
	) error

	UpdateSenha(
		ctx context.Context,
		tx *sql.Tx,
		model *Usuario,

	) error

	Delete(
		ctx context.Context,
		tx *sql.Tx,
		id uuid.UUID,
	) error
}

func NewRepository(
	db *sql.DB,
	logger jsonlog.Logger,
) UsuarioRepository {
	return &usuarioRepository{
		db:             db,
		logger:         logger,
		baseRepository: repository.NewBaseRepository[Usuario](db, logger, "usuarios", "u"),
	}
}

func parseUserConstraintError(err error) error {
	if pqErr, ok := err.(*pq.Error); ok {
		switch pqErr.Constraint {
		case "user_email_key":
			return e.ValidationAlreadyExists("email")
		case "user_telefone_key":
			return e.ValidationAlreadyExists("telefone")
		case "usuarios_cnpj_fkey":
			return e.ErrRecordNotFound
		}
	}
	return err
}

func (r *usuarioRepository) FindByEmail(
	ctx context.Context,
	email string,
) (*Usuario, error) {
	return r.baseRepository.FindOne(
		ctx,
		repository.WithQueryExtraWhere("u.email = :email", map[string]any{"email": email}),
	)
}

func (r *usuarioRepository) FindByID(
	ctx context.Context,
	ID uuid.UUID,
) (*Usuario, error) {
	user := contexts.ContextGetUser(ctx)
	params := map[string]any{
		"id":   ID,
		"cnpj": user.GetCNPJ(),
	}
	return r.baseRepository.FindOne(
		ctx,
		repository.WithQueryExtraWhere("u.id = :id and u.cnpj = :cnpj", params),
	)
}

func (r *usuarioRepository) FindAll(
	ctx context.Context,
	nome, telefone, email, cnpj string,
	f filters.Filters,
) ([]*Usuario, filters.Metadata, error) {
	params := map[string]any{
		"cnpj":     cnpj,
		"nome":     nome,
		"telefone": telefone,
		"email":    email,
	}
	query := fmt.Sprintf(`
			%s
			and u.cnpj = :cnpj
	`, repository.BuildFilterQuery("u", nome, telefone, email, cnpj))

	return r.baseRepository.FindWithFilters(
		ctx,
		f,
		repository.WithQueryExtraWhere(query, params),
	)
}

func (r *usuarioRepository) Insert(
	ctx context.Context,
	tx *sql.Tx,
	model *Usuario,
) error {
	err := r.baseRepository.Insert(
		ctx,
		tx,
		model,
	)

	if err != nil {
		return parseUserConstraintError(err)
	}
	return nil
}

func (r *usuarioRepository) Update(
	ctx context.Context,
	tx *sql.Tx,
	model *Usuario,
) error {
	user := contexts.ContextGetUser(ctx)

	err := r.baseRepository.Update(
		ctx,
		tx,
		model,
		model.ID,
		repository.WithExtraWhere("AND u.cnpj = :cnpj", map[string]any{
			"cnpj": user.GetCNPJ(),
		}),
	)

	if err != nil {
		return parseUserConstraintError(err)
	}
	return nil
}

func (r *usuarioRepository) UpdateSenha(
	ctx context.Context,
	tx *sql.Tx,
	model *Usuario,
) error {
	user := contexts.ContextGetUser(ctx)

	query := `
	update usuarios
	set
		password_hash = :senha,
		updated_at = now(),
		updated_by = :ID,
		version = usuarios.version + 1
	where
		id = :userId
    	and cnpj = :cnpj
		and version = :version
		and deleted = false
	`

	params := map[string]any{
		"senha":   model.Senha.Hash,
		"cnpj":    user.GetCNPJ(),
		"ID":      user.GetID(),
		"version": model.Version,
		"userId":  model.ID,
	}

	query, args := repository.NamedQuery(query, params)
	r.logger.PrintInfo(utils.MinifySQL(query), nil)

	result, err := tx.ExecContext(ctx, query, args...)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return e.ErrRecordNotFound
	}

	return nil
}

func (r *usuarioRepository) Delete(
	ctx context.Context,
	tx *sql.Tx,
	id uuid.UUID,
) error {
	user := contexts.ContextGetUser(ctx)

	params := map[string]any{
		"id":     id,
		"userID": user.GetID(),
		"cnpj":   user.GetCNPJ(),
	}

	query := `
		id = :id
		and cnpj = :cnpj
	`
	return r.baseRepository.DeleteByQuery(
		ctx,
		tx,
		repository.WithQueryExtraWhere(query, params),
	)
}
