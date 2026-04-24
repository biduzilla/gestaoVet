package repository

import (
	"context"
	"database/sql"
	"fmt"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/utils"
	"reflect"
	"strings"
)

type BaseRepository[T any] interface {
	FindById(ctx context.Context, id any, opts ...QueryOption) (*T, error)
	Find(ctx context.Context, query string, params map[string]any, opts ...QueryOption) ([]*T, error)
	FindOne(ctx context.Context, query string, params map[string]any, opts ...QueryOption) (*T, error)
	FindWithFilters(
		ctx context.Context,
		f filters.Filters,
		query string,
		params map[string]any,
		opts ...QueryOption,
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

type JoinSpec struct {
	Model     any
	Table     string
	Alias     string
	On        string
	FieldName string
}

type queryConfig struct {
	joins       []JoinSpec
	extraWhere  string
	extraParams map[string]any
}

type QueryOption func(*queryConfig)

func WithJoin(model any, table, alias, on, fieldName string) QueryOption {
	return func(c *queryConfig) {
		c.joins = append(c.joins, JoinSpec{
			Model:     model,
			Table:     table,
			Alias:     alias,
			On:        on,
			FieldName: fieldName,
		})
	}
}

func WithExtraWhereQuery(where string, params map[string]any) QueryOption {
	return func(c *queryConfig) {
		c.extraWhere = where
		c.extraParams = params
	}
}

func newQueryConfig(opts ...QueryOption) *queryConfig {
	cfg := &queryConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
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

func (r *baseRepository[T]) FindById(
	ctx context.Context,
	id any,
	opts ...QueryOption,
) (*T, error) {
	cfg := newQueryConfig(opts...)

	query := fmt.Sprintf(`
	select %s
	from %s as %s
	%s
	where
		id = $1
		and deleted = false
	`, r.selectColumns(cfg), r.table, r.alias, r.buildJoinClauses(cfg))

	r.logger.PrintInfo(utils.MinifySQL(query), nil)

	return GetByQuery[T](ctx, r.db, query, []any{id})
}

func (r *baseRepository[T]) Find(
	ctx context.Context,
	query string,
	params map[string]any,
	opts ...QueryOption,
) ([]*T, error) {
	cfg := newQueryConfig(opts...)

	finalQuery := fmt.Sprintf(`
	select %s
	from %s as %s
	%s
	where
		%s
		and deleted = false
	`, r.selectColumns(cfg), r.table, r.alias, r.buildJoinClauses(cfg), query)

	queryStr, args := NamedQuery(finalQuery, params)
	r.logger.PrintInfo(utils.MinifySQL(queryStr), nil)

	return ListQuery(ctx, r.db, queryStr, args, r.factory)
}

func (r *baseRepository[T]) FindOne(
	ctx context.Context,
	query string,
	params map[string]any,
	opts ...QueryOption,
) (*T, error) {
	cfg := newQueryConfig(opts...)

	finalQuery := fmt.Sprintf(`
	select %s
	from %s as %s
	%s
	where
		%s
		and deleted = false
	limit 1
	`, r.selectColumns(cfg), r.table, r.alias, r.buildJoinClauses(cfg), query)

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
	opts ...QueryOption,
) ([]*T, filters.Metadata, error) {
	cfg := newQueryConfig(opts...)

	finalQuery := fmt.Sprintf(`
        SELECT COUNT(*) OVER(), %s
        FROM %s as %s
		%s
        WHERE %s AND deleted = false
        ORDER BY %s %s
       	LIMIT :limit
        OFFSET :offset
    `,
		r.selectColumns(cfg),
		r.table,
		r.alias,
		r.buildJoinClauses(cfg),
		query,
		f.SortColumn(),
		f.SortDirection(),
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

func (r *baseRepository[T]) selectColumns(cfg *queryConfig) string {
	var model T
	return SelectColumns(cfg, model, r.alias)
}

func (r *baseRepository[T]) factory() *T {
	var model T
	return &model
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

func (r *baseRepository[T]) buildJoinClauses(cfg *queryConfig) string {
	if len(cfg.joins) == 0 {
		return ""
	}

	var joins []string
	for _, j := range cfg.joins {
		joins = append(joins, fmt.Sprintf("LEFT JOIN %s %s ON %s", j.Table, j.Alias, j.On))
	}
	return "\n" + strings.Join(joins, "\n")
}
