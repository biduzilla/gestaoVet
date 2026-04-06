package usuario

import (
	"context"
	"database/sql"
	"errors"
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
	db             *sql.DB
	logger         jsonlog.Logger
	baseRepository repository.BaseRepository[Usuario]
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
	email string,
) (*Usuario, error) {
	return r.baseRepository.FindOne("u.email = :email", map[string]any{"email": email})
}

func (r *usuarioRepository) FindByID(
	ID uuid.UUID,
	cnpj string,
) (*Usuario, error) {
	params := map[string]any{
		"id":   ID,
		"cnpj": cnpj,
	}
	return r.baseRepository.FindOne("u.id = :id and u.cnpj = :cnpj", params)
}

func (r *usuarioRepository) FindAll(
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
	return r.baseRepository.FindWithFilters(f, query, params)
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
	params := map[string]any{
		"id":     id,
		"userID": userID,
		"cnpj":   cnpj,
	}

	query := `
		id = :id
		and cnpj = :cnpj
	`
	return r.baseRepository.DeleteByQuery(tx, query, params)
}
