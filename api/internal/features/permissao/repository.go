package permissao

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

type CargoRepository interface {
	FindByID(id uuid.UUID, cnpj string) (*Cargo, error)
	FindAll(nome string, f filters.Filters, cnpj string) ([]*Cargo, filters.Metadata, error)
	Insert(tx *sql.Tx, model *Cargo, userID uuid.UUID) error
	Update(tx *sql.Tx, model *Cargo, cnpj string, userID uuid.UUID) error
	Delete(tx *sql.Tx, id, userID uuid.UUID, cnpj string) error
}

type cargoRepository struct {
	db     *sql.DB
	logger jsonlog.Logger
}

func NewCargoRepository(db *sql.DB, logger jsonlog.Logger) CargoRepository {
	return &cargoRepository{
		db:     db,
		logger: logger,
	}
}

func parseCargoConstraintError(err error) error {
	if pqErr, ok := err.(*pq.Error); ok {
		switch pqErr.Constraint {
		case "cargos_nome_cnpj_key":
			return e.ValidationAlreadyExists("nome")
		}
	}
	return err
}

func (r *cargoRepository) FindByID(
	id uuid.UUID,
	cnpj string,
) (*Cargo, error) {
	cols := repository.SelectColumns(Cargo{}, "c")
	query := fmt.Sprintf(`
		SELECT %s
		FROM cargos c
		WHERE c.id = :id
		  AND c.cnpj = :cnpj
		  AND c.deleted = false
	`, cols)

	params := map[string]any{
		"id":   id,
		"cnpj": cnpj,
	}

	query, args := repository.NamedQuery(query, params)
	r.logger.PrintInfo(utils.MinifySQL(query), nil)

	cargo, err := repository.GetByQuery[Cargo](
		r.db,
		query,
		args,
	)
	if err != nil {
		return nil, err
	}

	perms, err := r.findPermissoesByCargoID(cargo.ID)
	if err != nil {
		return nil, err
	}
	cargo.Permissoes = perms

	return cargo, nil
}

func (r *cargoRepository) FindAll(
	nome string,
	f filters.Filters,
	cnpj string,
) ([]*Cargo, filters.Metadata, error) {
	cols := repository.SelectColumns(Cargo{}, "c")
	query := fmt.Sprintf(`
		SELECT COUNT(*) OVER(),
		       %s
		FROM cargos c
		WHERE (to_tsvector('simple', c.nome) @@ plainto_tsquery('simple', :nome) OR :nome = '')
		  AND c.cnpj = :cnpj
		  AND c.deleted = false
		ORDER BY c.%s %s, c.id ASC
		LIMIT :limit
		OFFSET :offset
	`, cols, f.SortColumn(), f.SortDirection())

	params := map[string]any{
		"cnpj":   cnpj,
		"nome":   nome,
		"limit":  f.Limit(),
		"offset": f.Offset(),
	}

	namedQuery, args := repository.NamedQuery(query, params)
	r.logger.PrintInfo(utils.MinifySQL(namedQuery), nil)

	cargos, metadata, err := repository.PaginatedQuery(
		r.db,
		namedQuery,
		args,
		f,
		func() *Cargo { return &Cargo{} },
	)

	if err != nil {
		return nil, metadata, err
	}

	if len(cargos) == 0 {
		return cargos, metadata, nil
	}

	cargosIDs := make([]uuid.UUID, len(cargos))
	for i, c := range cargos {
		cargosIDs[i] = c.ID
	}
	permsMap, err := r.findPermissoesByCargoIDs(cargosIDs)
	if err != nil {
		return nil, metadata, err
	}

	for _, cargo := range cargos {
		if perms, ok := permsMap[cargo.ID]; ok {
			cargo.Permissoes = perms
		}
	}

	return cargos, metadata, nil
}

func (r *cargoRepository) findPermissoesByCargoID(
	cargoID uuid.UUID,
) ([]Permissao, error) {
	query := `
		SELECT p.id, p.recurso, p.acao
		FROM permissoes p
		INNER JOIN cargo_permissoes cp ON cp.permissao_id = p.id
		WHERE cp.cargo_id = $1
	`
	r.logger.PrintInfo(utils.MinifySQL(query), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, query, cargoID)
	if err != nil {
		return nil, err
	}

	var permissoes []Permissao
	for rows.Next() {
		var p Permissao
		err := rows.Scan(&p.ID, &p.Recurso, &p.Acao)
		if err != nil {
			return nil, err
		}
		permissoes = append(permissoes, p)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return permissoes, nil
}

func (r *cargoRepository) findPermissoesByCargoIDs(
	cargoIDs []uuid.UUID,
) (map[uuid.UUID][]Permissao, error) {
	if len(cargoIDs) == 0 {
		return nil, nil
	}

	query := `
		SELECT cp.cargo_id, p.id, p.recurso, p.acao
		FROM permissoes p
		INNER JOIN cargo_permissoes cp ON cp.permissao_id = p.id
		WHERE cp.cargo_id = ANY($1)
	`
	r.logger.PrintInfo(utils.MinifySQL(query), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, query, pq.Array(cargoIDs))
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	permsMap := make(map[uuid.UUID][]Permissao)
	for rows.Next() {
		var cargoID uuid.UUID
		var p Permissao
		err := rows.Scan(
			&cargoID,
			&p.ID,
			&p.Recurso,
			&p.Acao,
		)
		if err != nil {
			return nil, err
		}
		permsMap[cargoID] = append(permsMap[cargoID], p)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return permsMap, nil
}

func (r *cargoRepository) Insert(
	tx *sql.Tx,
	model *Cargo,
	userID uuid.UUID,
) error {
	queryCargo := `
		INSERT INTO cargos (nome, cnpj,created_by)
		VALUES (:nome, :cnpj, :userID)
		RETURNING id, created_at, version
	`
	params := map[string]any{
		"nome":   model.Nome,
		"cnpj":   model.Cnpj,
		"userID": userID,
	}

	query, args := repository.NamedQuery(queryCargo, params)
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
		return parseCargoConstraintError(err)
	}

	if len(model.Permissoes) > 0 {
		return r.syncPermissoes(tx, model.ID, model.Permissoes)
	}

	return nil
}

func (r *cargoRepository) Update(
	tx *sql.Tx,
	model *Cargo,
	cnpj string,
	userID uuid.UUID,
) error {
	queryCargo := `
		UPDATE cargos
		SET nome = :nome,
		    updated_at = now(),
		    updated_by = :userID,
		    version = cargos.version + 1
		WHERE id = :id
		  AND cnpj = :cnpj
		  AND deleted = false
		  AND version = :version
		RETURNING version
	`

	params := map[string]any{
		"id":      model.ID,
		"nome":    model.Nome,
		"cnpj":    cnpj,
		"userID":  userID,
		"version": model.Version,
	}

	query, args := repository.NamedQuery(queryCargo, params)
	r.logger.PrintInfo(query, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := tx.QueryRowContext(ctx, query, args...).Scan(
		&model.Version,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return e.ErrEditConflict
		}
		return parseCargoConstraintError(err)
	}

	return r.syncPermissoes(tx, model.ID, model.Permissoes)
}

func (r *cargoRepository) syncPermissoes(
	tx *sql.Tx,
	cargoID uuid.UUID,
	permissoes []Permissao,
) error {
	if err := r.validatePermissoesExistem(permissoes); err != nil {
		return err
	}

	queryDelete := `
		delete from cargo_permissoes
		where cargo_id = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := tx.ExecContext(ctx, queryDelete, cargoID)
	if err != nil {
		return err
	}

	if len(permissoes) == 0 {
		return nil
	}

	queryInsert := `
		INSERT INTO cargo_permissoes (cargo_id, permissao_id)
		VALUES ($1, $2)
	`
	stmt, err := tx.PrepareContext(ctx, queryInsert)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range permissoes {
		_, err := stmt.ExecContext(ctx, cargoID, p.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *cargoRepository) Delete(tx *sql.Tx,
	id, userID uuid.UUID,
	cnpj string,
) error {
	query := `
		update cargos
		set deleted = true,
			updated_at = now(),
			updated_by = :userID
		where id = :id
			and cnpj = :cnpj
			and deleted = false
		
	`
	params := map[string]any{
		"id":     id,
		"cnpj":   cnpj,
		"userID": userID,
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

func (r *cargoRepository) validatePermissoesExistem(
	permissoes []Permissao,
) error {
	if len(permissoes) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, len(permissoes))
	for i, p := range permissoes {
		ids[i] = p.ID
	}

	query := `
		select 
			count (*) 
		from permissoes 
			where id = any($1)`

	var count int
	r.logger.PrintInfo(utils.MinifySQL(query), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := r.db.QueryRowContext(ctx, query, pq.Array(ids)).Scan(&count)
	if err != nil {
		return err
	}

	if count != len(ids) {
		return e.ErrCountPermissions
	}

	return nil
}
