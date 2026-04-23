package usuario

import (
	"context"
	"database/sql"
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
	return r.baseRepository.FindOne(ctx, "u.email = :email", map[string]any{"email": email})
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
	return r.baseRepository.FindOne(ctx, "u.id = :id and u.cnpj = :cnpj", params)
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
	query := `
			(to_tsvector('simple', u.nome) @@ plainto_tsquery('simple', :nome) OR :nome = '')
            and (to_tsvector('simple', u.telefone) @@ plainto_tsquery('simple', :telefone) OR :telefone = '')
            and (to_tsvector('simple', u.email) @@ plainto_tsquery('simple', :email) OR :email = '') 
			and u.deleted = false
			and u.cnpj = :cnpj
	`
	return r.baseRepository.FindWithFilters(ctx, f, query, params)
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
	// query := `
	// insert into usuarios (
	// 	nome,
	// 	telefone,
	// 	email,
	// 	password_hash,
	// 	cnpj,
	// 	roles
	// )
	// values (
	// 	:nome,
	// 	:telefone,
	// 	:email,
	// 	:senha,
	// 	:cnpj,
	// 	:roles
	// )
	// returning
	// 	id,
	// 	created_at,
	// 	version
	// `

	// params := map[string]any{
	// 	"nome":     model.Nome,
	// 	"telefone": model.Telefone,
	// 	"cnpj":     model.Cnpj,
	// 	"email":    model.Email,
	// 	"senha":    model.Senha.Hash,
	// 	"roles":    model.Roles,
	// }

	// query, args := repository.NamedQuery(query, params)

	// r.logger.PrintInfo(utils.MinifySQL(query), nil)

	// err := tx.QueryRowContext(ctx, query, args...).Scan(
	// 	&model.ID,
	// 	&model.CreatedAt,
	// 	&model.Version,
	// )

	// if err != nil {
	// 	if errors.Is(err, sql.ErrNoRows) {
	// 		return e.ErrRecordNotFound
	// 	}

	// 	return parseUserConstraintError(err)
	// }

	// return nil
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
		repository.WithExtraWhere("AND cnpj = :cnpj", map[string]any{
			"cnpj": user.GetCNPJ(),
		}),
	)

	if err != nil {
		return parseUserConstraintError(err)
	}
	return nil

	// query := `
	// update usuarios
	// set
	// 	nome = :nome,
	// 	telefone = :telefone,
	// 	email = :email,
	// 	updated_at = now(),
	// 	updated_by = :ID,
	// 	roles = :roles
	// 	version = usuarios.version + 1
	// where
	// 	id = :userId
	// 	and cnpj = :cnpj
	// 	and version = :version
	// 	and deleted = false
	// returning
	// 	version
	// `

	// params := map[string]any{
	// 	"nome":     model.Nome,
	// 	"telefone": model.Telefone,
	// 	"cnpj":     user.GetCNPJ(),
	// 	"email":    model.Email,
	// 	"ID":       user.GetID(),
	// 	"version":  model.Version,
	// 	"roles":    model.Roles,
	// 	"userId":   model.ID,
	// }

	// query, args := repository.NamedQuery(query, params)
	// r.logger.PrintInfo(utils.MinifySQL(query), nil)

	// err := tx.QueryRowContext(ctx, query, args...).Scan(
	// 	&model.Version,
	// )

	// if err != nil {
	// 	if errors.Is(err, sql.ErrNoRows) {
	// 		return e.ErrEditConflict
	// 	}

	// 	return parseUserConstraintError(err)
	// }

	// return nil
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
	return r.baseRepository.DeleteByQuery(ctx, tx, query, params)
}
