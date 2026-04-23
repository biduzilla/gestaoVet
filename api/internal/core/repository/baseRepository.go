package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/utils"
	"maps"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type MutationOption func(*mutationConfig)

type mutationConfig struct {
	skipAudit    bool
	ignoreFields []string
	extraWhere   string
	extraParams  map[string]any
}

func WithSkipAudit() MutationOption {
	return func(mc *mutationConfig) { mc.skipAudit = true }
}

func WithExtraWhere(where string, params map[string]any) MutationOption {
	return func(mc *mutationConfig) {
		mc.extraWhere = where
		mc.extraParams = params
	}
}

type BaseRepository[T any] interface {
	FindById(ctx context.Context, id any) (*T, error)
	Find(ctx context.Context, query string, params map[string]any) ([]*T, error)
	FindOne(ctx context.Context, query string, params map[string]any) (*T, error)
	FindWithFilters(
		ctx context.Context,
		f filters.Filters,
		query string,
		params map[string]any,
	) ([]*T, filters.Metadata, error)
	DeleteByQuery(ctx context.Context, tx *sql.Tx, query string, params map[string]any) error
	Count(ctx context.Context, query string, params map[string]any) (int64, error)
	Exists(ctx context.Context, query string, params map[string]any) (bool, error)
	Insert(ctx context.Context, tx *sql.Tx, model *T, opts ...MutationOption) error
	Update(ctx context.Context, tx *sql.Tx, model *T, id any, opts ...MutationOption) error
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
	tableName string,
	alias string,
) BaseRepository[T] {
	return &baseRepository[T]{
		db:     db,
		logger: logger,
		table:  tableName,
		alias:  alias,
	}
}

func (r *baseRepository[T]) Exists(ctx context.Context, query string, params map[string]any) (bool, error) {
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
	err := r.db.QueryRowContext(ctx, namedQuery, args...).Scan(&exists)
	return exists, err
}

func (r *baseRepository[T]) FindById(ctx context.Context, id any) (*T, error) {
	query := fmt.Sprintf(`
	select %s
	from %s
	where
		id = $1
		and deleted = false
	`, r.selectColumns(), r.table)

	r.logger.PrintInfo(utils.MinifySQL(query), nil)

	return GetByQuery[T](ctx, r.db, query, []any{id})
}

func (r *baseRepository[T]) Find(ctx context.Context, query string, params map[string]any) ([]*T, error) {
	finalQuery := fmt.Sprintf(`
	select %s
	from %s as %s
	where
		%s
		and deleted = false
	`, r.selectColumns(), r.table, r.alias, query)

	queryStr, args := NamedQuery(finalQuery, params)
	r.logger.PrintInfo(utils.MinifySQL(queryStr), nil)

	return ListQuery(ctx, r.db, queryStr, args, r.factory)
}

func (r *baseRepository[T]) FindOne(ctx context.Context, query string, params map[string]any) (*T, error) {
	finalQuery := fmt.Sprintf(`
	select %s
	from %s as %s
	where
		%s
		and deleted = false
	limit 1
	`, r.selectColumns(), r.table, r.alias, query)

	queryStr, args := NamedQuery(finalQuery, params)
	r.logger.PrintInfo(utils.MinifySQL(queryStr), nil)

	return GetByQuery[T](ctx, r.db, queryStr, args)
}

func (r *baseRepository[T]) Count(ctx context.Context, query string, params map[string]any) (int64, error) {
	finalQuery := fmt.Sprintf(`
        SELECT COUNT(*)
        FROM %s as %s
        WHERE %s AND deleted = false
    `, r.table, r.alias, query)

	queryStr, args := NamedQuery(finalQuery, params)
	r.logger.PrintInfo(utils.MinifySQL(queryStr), nil)

	var count int64
	err := r.db.QueryRowContext(ctx, queryStr, args...).Scan(&count)
	return count, err
}

func (r *baseRepository[T]) FindWithFilters(
	ctx context.Context,
	f filters.Filters,
	query string,
	params map[string]any,
) ([]*T, filters.Metadata, error) {
	finalQuery := fmt.Sprintf(`
        SELECT COUNT(*) OVER(), %s
        FROM %s as %s
        WHERE %s AND deleted = false
        ORDER BY %s %s
       	LIMIT :limit
        OFFSET :offset
    `, r.selectColumns(), r.table, r.alias, query, f.SortColumn(), f.SortDirection(),
	)

	params["limit"] = f.Limit()
	params["offset"] = f.Offset()

	queryStr, args := NamedQuery(finalQuery, params)
	r.logger.PrintInfo(utils.MinifySQL(queryStr), nil)

	return PaginatedQuery(ctx, r.db, queryStr, args, f, r.factory)
}

func (r *baseRepository[T]) DeleteByQuery(ctx context.Context, tx *sql.Tx, query string, params map[string]any) error {
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
	return SelectColumns(model, r.alias)
}

func (r *baseRepository[T]) factory() *T {
	var model T
	return &model
}

func (r *baseRepository[T]) Insert(
	ctx context.Context,
	tx *sql.Tx,
	model *T,
	opts ...MutationOption,
) error {
	cfg := &mutationConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	params, err := CollectParams(model, "insert")
	if err != nil {
		return fmt.Errorf("collect params: %w", err)
	}

	values := ParamsToMap(params)
	maps.Copy(values, cfg.extraParams)

	if !cfg.skipAudit {
		values["created_at"] = time.Now()
	}

	query := BuildInsertQuery(r.table, params, []string{"id", "created_at", "version"})
	queryStr, args := NamedQuery(query, values)
	r.logger.PrintInfo(utils.MinifySQL(queryStr), nil)

	type returning struct {
		ID        uuid.UUID `db:"id"`
		CreatedAt time.Time `db:"created_at"`
		Version   int       `db:"version"`
	}
	var ret returning

	err = tx.QueryRowContext(ctx, queryStr, args...).Scan(
		&ret.ID,
		&ret.CreatedAt,
		&ret.Version,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return e.ErrRecordNotFound
		}
		return parseConstraintError(err)
	}

	r.setFieldValue(model, "ID", ret.ID)
	r.setFieldValue(model, "CreatedAt", ret.CreatedAt)
	r.setFieldValue(model, "Version", ret.Version)

	return nil
}

func (r *baseRepository[T]) Update(
	ctx context.Context,
	tx *sql.Tx,
	model *T,
	id any,
	opts ...MutationOption,
) error {
	cfg := &mutationConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	params, err := CollectParams(model, "update")
	if err != nil {
		return fmt.Errorf("collect params: %w", err)
	}

	values := ParamsToMap(params)
	values["id"] = id

	maps.Copy(values, cfg.extraParams)

	if !cfg.skipAudit {
		values["updated_at"] = time.Now()
	}

	extraWhere := cfg.extraWhere
	if extraWhere != "" {
		extraWhere += " AND version = :version"
	} else {
		extraWhere = "version = :version"
	}
	values["version"] = r.getFieldValue(model, "Version")

	query := BuildUpdateQuery(r.table, params, "id", extraWhere, []string{"version"})
	queryStr, args := NamedQuery(query, values)
	r.logger.PrintInfo(utils.MinifySQL(queryStr), nil)

	var newVersion int
	err = tx.QueryRowContext(ctx, queryStr, args...).Scan(&newVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return e.ErrEditConflict
		}
		return parseConstraintError(err)
	}

	r.setFieldValue(model, "Version", newVersion)

	return nil
}

func (r *baseRepository[T]) setFieldValue(
	model *T,
	fieldName string,
	value any,
) {
	v := reflect.ValueOf(model).Elem()
	field := v.FieldByName(fieldName)
	if field.IsValid() && field.CanSet() {
		field.Set(reflect.ValueOf(value))
	}
}

func (r *baseRepository[T]) getFieldValue(model *T, fieldName string) any {
	v := reflect.ValueOf(model).Elem()
	field := v.FieldByName(fieldName)
	if field.IsValid() {
		return field.Interface()
	}
	return nil
}

func parseConstraintError(err error) error {
	if pqErr, ok := err.(*pq.Error); ok {
		if strings.Contains(pqErr.Constraint, "_key") {
			return e.ValidationAlreadyExists("campo")
		}
	}
	return err
}
