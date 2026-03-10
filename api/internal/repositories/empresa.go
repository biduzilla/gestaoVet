package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"gestaoVet/internal/jsonlog"
	"gestaoVet/internal/models"
	"gestaoVet/internal/models/filters"
	"gestaoVet/utils"
	"gestaoVet/utils/errors"
	e "gestaoVet/utils/errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type empresaRepository struct {
	db     *sql.DB
	logger jsonlog.Logger
}

func NewEmpresaRepository(
	db *sql.DB,
	logger jsonlog.Logger,
) *empresaRepository {
	return &empresaRepository{
		db:     db,
		logger: logger,
	}
}

type EmpresaRepository interface {
	FindByID(
		ID uuid.UUID,
	) (*models.Empresa, error)

	FindAll(
		cnpj, nomeFantasia, razaoSocial, email string,
		f filters.Filters,
	) ([]*models.Empresa, filters.Metadata, error)

	InsertOrUpdate(
		tx *sql.Tx,
		model *models.Empresa,
	) error

	Delete(tx *sql.Tx, id, userID uuid.UUID) error
}

func parseUserConstraintError(err error) error {
	if pqErr, ok := err.(*pq.Error); ok {
		switch pqErr.Constraint {
		case "empresas_email_key":
			return errors.ValidationAlreadyExists("email")
		case "empresas_razao_social_key":
			return errors.ValidationAlreadyExists("razao_social")
		case "empresas_nome_fantasia_key":
			return errors.ValidationAlreadyExists("nome_fantasia")
		case "empresas_cnpj_key":
			return errors.ValidationAlreadyExists("cnpj")
		}
	}
	return err
}

func (r *empresaRepository) FindByID(
	ID uuid.UUID,
) (*models.Empresa, error) {
	cols := selectColumns(models.Empresa{}, "e")
	query := fmt.Sprintf(`
		select
			%s
		from empresas e
		where
			e.id = :id
			and deleted = false
	`, cols)

	params := map[string]any{
		"id": ID,
	}

	query, args := namedQuery(query, params)
	r.logger.PrintInfo(utils.MinifySQL(query), nil)

	return getByQuery[models.Empresa](r.db, query, args)
}

func (r *empresaRepository) FindAll(
	cnpj, nomeFantasia, razaoSocial, email string,
	f filters.Filters,
) ([]*models.Empresa, filters.Metadata, error) {
	cols := selectColumns(models.Empresa{}, "e")

	query := fmt.Sprintf(`
		select
			count(*) OVER(),
			%s
		from empresas e
        WHERE
            (to_tsvector('simple', e.cnpj) @@ plainto_tsquery('simple', :cnpj) OR :cnpj = '')
            (to_tsvector('simple', e.nomeFantasia) @@ plainto_tsquery('simple', :nomeFantasia) OR :nomeFantasia = '')
            (to_tsvector('simple', e.razaoSocial) @@ plainto_tsquery('simple', :razaoSocial) OR :razaoSocial = '')
            (to_tsvector('simple', e.email) @@ plainto_tsquery('simple', :email) OR :email = '')
			where e.deleted = false
		ORDER BY
            e.%s %s,
            e.id ASC
        LIMIT :limit
        OFFSET :offset
	`, cols,
		f.SortColumn(),
		f.SortDirection())

	params := map[string]any{
		"cnpj":         cnpj,
		"nomeFantasia": nomeFantasia,
		"razaoSocial":  razaoSocial,
		"email":        email,
		"limit":        f.Limit(),
		"offset":       f.Offset(),
	}

	query, args := namedQuery(query, params)
	r.logger.PrintInfo(utils.MinifySQL(query), nil)

	return paginatedQuery(
		r.db,
		query,
		args,
		f,
		func() *models.Empresa {
			return &models.Empresa{}
		},
	)
}

func (r *empresaRepository) InsertOrUpdate(
	tx *sql.Tx,
	model *models.Empresa,
) error {
	query := `
	insert into empresas (
		nome_fantasia,
		razao_social,
		cnpj,
		email,
	)
	values (
		:nomeFantasia,
		:razaoSocial,
		:cnpj,
		:email
	)
	on conflict (
		nome_fantasia,
		razao_social,
		cnpj,
		email
	) where deleted = false
	do update set
		nome_fantasia = excluded.nome_fantasia,
		razao_social = excluded.razao_social,
		email = excluded.email,
		updated_at = now()
		updated_by = empresas.id,
		version = empresas.version + 1
	returning
		id,
		created_at,
		version,
	`

	params := map[string]any{
		"nomeFantasia": model.NomeFantasia,
		"razaoSocial":  model.RazaoSocial,
		"cnpj":         model.Cnpj,
		"email":        model.Email,
	}

	query, args := namedQuery(query, params)
	r.logger.PrintInfo(utils.MinifySQL(query), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return tx.QueryRowContext(ctx, query, args...).Scan(
		&model.ID,
		&model.CreatedAt,
		&model.Version,
	)
}

func (r *empresaRepository) Delete(tx *sql.Tx, id, userID uuid.UUID) error {
	query := `
	UPDATE empresas set
		deleted = true
		updated_at = now()
		updated_by = :userID
	where 
		id = :id
	`

	params := map[string]any{
		"id":     id,
		"userID": userID,
	}

	query, args := namedQuery(query, params)
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
