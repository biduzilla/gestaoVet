package usuario

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/internal/core/repository"
	"gestaoVet/utils"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type usuarioRepository struct {
	db     *sql.DB
	logger jsonlog.Logger
}

type UsuarioRepository interface {
	FindByEmail(
		email string,
	) (*Usuario, error)

	FindByID(
		ID uuid.UUID,
		cnpj string,
	) (*Usuario, error)

	FindAll(
		nome, telefone, email, cnpj string,
		f filters.Filters,
	) ([]*Usuario, filters.Metadata, error)

	Insert(
		tx *sql.Tx,
		model *Usuario,
	) error

	Update(
		tx *sql.Tx,
		model *Usuario,
		cnpj string,
		ID uuid.UUID,
	) error

	Delete(tx *sql.Tx,
		id,
		userID uuid.UUID,
		cnpj string,
	) error
}

func NewRepository(
	db *sql.DB,
	logger jsonlog.Logger,
) *usuarioRepository {
	return &usuarioRepository{
		db:     db,
		logger: logger,
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
	email string,
) (*Usuario, error) {
	cols := repository.SelectColumns(Usuario{}, "u")
	query := fmt.Sprintf(`
		select
			%s
		from usuarios u
		where
			u.email = :email
			and u.deleted = false
	`, cols)

	params := map[string]any{
		"email": email,
	}

	query, args := repository.NamedQuery(query, params)
	r.logger.PrintInfo(utils.MinifySQL(query), nil)

	return repository.GetByQuery[Usuario](r.db, query, args)
}

func (r *usuarioRepository) FindByID(
	ID uuid.UUID,
	cnpj string,
) (*Usuario, error) {
	cols := repository.SelectColumns(Usuario{}, "u")
	query := fmt.Sprintf(`
		select
			%s
		from usuarios u
		where
			u.id = :id
			and u.cnpj = :cnpj
			and u.deleted = false
	`, cols)

	params := map[string]any{
		"id":   ID,
		"cnpj": cnpj,
	}

	query, args := repository.NamedQuery(query, params)
	r.logger.PrintInfo(utils.MinifySQL(query), nil)

	return repository.GetByQuery[Usuario](r.db, query, args)
}

func (r *usuarioRepository) FindAll(
	nome, telefone, email, cnpj string,
	f filters.Filters,
) ([]*Usuario, filters.Metadata, error) {
	cols := repository.SelectColumns(Usuario{}, "u")

	query := fmt.Sprintf(`
		select
			count(*) OVER(),
			%s
		from usuarios u
        WHERE
            (to_tsvector('simple', u.nome) @@ plainto_tsquery('simple', :nome) OR :nome = '')
            (to_tsvector('simple', u.telefone) @@ plainto_tsquery('simple', :telefone) OR :telefone = '')
            (to_tsvector('simple', u.email) @@ plainto_tsquery('simple', :email) OR :email = '')
			where 
				u.deleted = false
				and u.cnpj = :cnpj
		ORDER BY
            u.%s %s,
            u.id ASC
        LIMIT :limit
        OFFSET :offset
	`, cols,
		f.SortColumn(),
		f.SortDirection())

	params := map[string]any{
		"cnpj":     cnpj,
		"nome":     nome,
		"telefone": telefone,
		"email":    email,
		"limit":    f.Limit(),
		"offset":   f.Offset(),
	}

	query, args := repository.NamedQuery(query, params)
	r.logger.PrintInfo(utils.MinifySQL(query), nil)

	return repository.PaginatedQuery(
		r.db,
		query,
		args,
		f,
		func() *Usuario {
			return &Usuario{}
		},
	)
}

func (r *usuarioRepository) Insert(
	tx *sql.Tx,
	model *Usuario,
) error {
	query := `
	insert into usuarios (
		nome,
		telefone,
		email,
		password_hash,
		cnpj
	)
	values (
		:nome,
		:telefone,
		:email,
		:senha,
		:cnpj
	)
	returning
		id,
		created_at,
		version
	`

	params := map[string]any{
		"nome":     model.Nome,
		"telefone": model.Telefone,
		"cnpj":     model.Cnpj,
		"email":    model.Email,
		"senha":    model.Senha.Hash,
	}

	query, args := repository.NamedQuery(query, params)
	r.logger.PrintInfo(utils.MinifySQL(query), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := tx.QueryRowContext(ctx, query, args...).Scan(
		&model.ID,
		&model.CreatedAt,
		&model.Version,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return e.ErrRecordNotFound
		}

		return parseUserConstraintError(err)
	}

	return nil
}

func (r *usuarioRepository) Update(
	tx *sql.Tx,
	model *Usuario,
	cnpj string,
	ID uuid.UUID,
) error {
	query := `
	update usuarios
	set
		nome = :nome,
		telefone = :telefone,
		email = :email,
		updated_at = now(),
		updated_by = :ID,
		version = usuarios.version + 1
	where
    	cnpj = :cnpj
		version = :version
		and deleted = false
	returning
		version
	`

	params := map[string]any{
		"nome":     model.Nome,
		"telefone": model.Telefone,
		"cnpj":     cnpj,
		"email":    model.Email,
		"ID":       ID,
		"version":  model.Version,
	}

	query, args := repository.NamedQuery(query, params)
	r.logger.PrintInfo(utils.MinifySQL(query), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := tx.QueryRowContext(ctx, query, args...).Scan(
		&model.Version,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return e.ErrEditConflict
		}

		return parseUserConstraintError(err)
	}

	return nil
}

func (r *usuarioRepository) Delete(tx *sql.Tx, id, userID uuid.UUID, cnpj string) error {
	query := `
	UPDATE usuarios set
		deleted = true,
		updated_at = now(),
		updated_by = :userID
	where 
		id = :id
		and cnpj = :cnpj
		and deleted = false
	`

	params := map[string]any{
		"id":     id,
		"userID": userID,
		"cnpj":   cnpj,
	}

	query, args := repository.NamedQuery(query, params)
	r.logger.PrintInfo(utils.MinifySQL(query), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

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
