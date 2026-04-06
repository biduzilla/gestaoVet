package repository

import (
	"context"
	"database/sql"
	"fmt"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/utils"
	"strings"
	"time"
)

type BaseRepository[T any] interface {
	FindById(id any) (*T, error)
	Find(query string, params map[string]any) ([]*T, error)
	FindOne(query string, params map[string]any) (*T, error)
	FindWithFilters(
		f filters.Filters,
		query string,
		params map[string]any,
	) ([]*T, filters.Metadata, error)
	DeleteByQuery(tx *sql.Tx, query string, params map[string]any) error
	Count(query string, params map[string]any) (int64, error)
	Exists(query string, params map[string]any) (bool, error)
}

type baseRepository[T any] struct {
	db     *sql.DB
	logger jsonlog.Logger
	table  string
	alias  string
}

func NewBaseRepository[T any](
	db *sql.DB,
	logger jsonlog.Logger,
	alias string,
) BaseRepository[T] {
	var table T
	tableName := utils.GetTypeName(table)

	return &baseRepository[T]{
		db:     db,
		logger: logger,
		table:  tableName,
		alias:  alias,
	}
}

func (r *baseRepository[T]) Exists(query string, params map[string]any) (bool, error) {
	if strings.TrimSpace(query) == "" {
		query = "1 = 1"
	}

	queryStr := fmt.Sprintf(`
        SELECT EXISTS (
            SELECT 1 
            FROM %s 
            WHERE %s AND deleted = false
        )
    `, r.table, query)

	namedQuery, args := NamedQuery(queryStr, params)
	r.logger.PrintInfo(utils.MinifySQL(namedQuery), nil)

	var exists bool
	err := r.db.QueryRow(namedQuery, args...).Scan(&exists)
	return exists, err
}

func (r *baseRepository[T]) FindById(id any) (*T, error) {
	query := fmt.Sprintf(`
	select %s
	from %s
	where
		id = $1
		and deleted = false
	`, r.selectColumns(), r.table)

	r.logger.PrintInfo(utils.MinifySQL(query), nil)

	return GetByQuery[T](r.db, query, []any{id})
}

func (r *baseRepository[T]) Find(query string, params map[string]any) ([]*T, error) {
	finalQuery := fmt.Sprintf(`
	select %s
	from %s as %s
	where
		%s
		and deleted = false
	`, r.selectColumns(), r.alias, r.table, query)

	queryStr, args := NamedQuery(finalQuery, params)
	r.logger.PrintInfo(utils.MinifySQL(queryStr), nil)

	return ListQuery(r.db, queryStr, args, r.factory)
}

func (r *baseRepository[T]) FindOne(query string, params map[string]any) (*T, error) {
	finalQuery := fmt.Sprintf(`
	select %s
	from %s as %s
	where
		%s
		and deleted = false
	limit 1
	`, r.selectColumns(), r.alias, r.table, query)

	queryStr, args := NamedQuery(finalQuery, params)
	r.logger.PrintInfo(utils.MinifySQL(queryStr), nil)

	return GetByQuery[T](r.db, queryStr, args)
}

func (r *baseRepository[T]) Count(query string, params map[string]any) (int64, error) {
	finalQuery := fmt.Sprintf(`
        SELECT COUNT(*)
        FROM %s as %s
        WHERE %s AND deleted = false
    `, r.table, r.alias, query)

	queryStr, args := NamedQuery(finalQuery, params)
	r.logger.PrintInfo(utils.MinifySQL(queryStr), nil)

	var count int64
	err := r.db.QueryRow(queryStr, args...).Scan(&count)
	return count, err
}

func (r *baseRepository[T]) FindWithFilters(
	f filters.Filters,
	query string,
	params map[string]any,
) ([]*T, filters.Metadata, error) {
	finalQuery := fmt.Sprintf(`
        SELECT COUNT(*) OVER(), %s
        FROM %s as %s
        WHERE %s AND deleted = false
        ORDER BY %s %s
        LIMIT $%d OFFSET $%d
    `, r.selectColumns(), r.table, r.alias, query, f.SortColumn(), f.SortDirection(),
		len(params)+1, len(params)+2)

	params["limit"] = f.PageSize
	params["offset"] = (f.Page - 1) * f.PageSize

	queryStr, args := NamedQuery(finalQuery, params)
	r.logger.PrintInfo(utils.MinifySQL(queryStr), nil)

	return PaginatedQuery(r.db, queryStr, args, f, r.factory)
}

func (r *baseRepository[T]) DeleteByQuery(tx *sql.Tx, query string, params map[string]any) error {
	finalQuery := fmt.Sprintf(`
	UPDATE %s set
		deleted = true,
		updated_at = now(),
		updated_by = :userID
	where 
		%s
		and deleted = false
	`, r.table, query)

	queryStr, args := NamedQuery(finalQuery, params)
	r.logger.PrintInfo(utils.MinifySQL(queryStr), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := tx.ExecContext(ctx, queryStr, args...)
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

func (r *baseRepository[T]) selectColumns() string {
	var model T
	return SelectColumns(model, r.table)
}

func (r *baseRepository[T]) factory() *T {
	var model T
	return &model
}
