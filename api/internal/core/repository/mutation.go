package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gestaoVet/internal/core/contexts"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/utils"
	"maps"
	"reflect"
	"strings"
	"time"
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

type pkInfo struct {
	columnName string
	isAuto     bool
	fieldName  string
}

func (r *baseRepository[T]) getPkInfo(model *T) (pkInfo, error) {
	v := reflect.ValueOf(model).Elem()
	t := v.Type()

	for field := range t.Fields() {
		repoTag := field.Tag.Get("repo")
		dbTag := field.Tag.Get("db")

		if strings.Contains(repoTag, "pk") && dbTag != "" {
			return pkInfo{
				columnName: dbTag,
				isAuto:     strings.Contains(repoTag, "auto"),
				fieldName:  field.Name,
			}, nil
		}

		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			if pk, err := r.getPkInfoFromStruct(field.Type); err == nil && pk.columnName != "" {
				return pk, nil
			}
		}
	}

	return pkInfo{columnName: "id", isAuto: true, fieldName: "ID"}, nil
}

func (r *baseRepository[T]) getPkInfoFromStruct(t reflect.Type) (pkInfo, error) {
	for field := range t.Fields() {
		field := field
		repoTag := field.Tag.Get("repo")
		dbTag := field.Tag.Get("db")

		if strings.Contains(repoTag, "pk") && dbTag != "" {
			return pkInfo{
				columnName: dbTag,
				isAuto:     strings.Contains(repoTag, "auto"),
				fieldName:  field.Name,
			}, nil
		}
	}
	return pkInfo{}, fmt.Errorf("pk not found in embedded struct")
}

func filterValuesForQuery(values map[string]any, query string) map[string]any {
	filtered := make(map[string]any, len(values))
	for k, v := range values {
		if v == nil {
			continue
		}
		if !strings.Contains(query, ":"+k) {
			continue
		}
		filtered[k] = v
	}
	return filtered
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
		user := contexts.ContextGetUser(ctx)
		maps.Copy(values, map[string]any{
			"created_at": time.Now(),
			"created_by": user.GetID()})
	}

	pk, err := r.getPkInfo(model)
	if err != nil {
		return fmt.Errorf("get pk info: %w", err)
	}

	var returningFields []string
	if !pk.isAuto {
		returningFields = append(returningFields, pk.columnName)
	}
	returningFields = append(returningFields, "created_at", "version")

	query := BuildInsertQuery(r.table, params, returningFields, cfg)

	filteredValues := filterValuesForQuery(values, query)
	queryStr, args := NamedQuery(query, filteredValues)
	r.logger.PrintInfo(utils.MinifySQL(queryStr), nil)

	if pk.isAuto {
		var ret struct {
			CreatedAt time.Time `db:"created_at"`
			Version   int       `db:"version"`
		}

		err = tx.QueryRowContext(ctx, queryStr, args...).Scan(&ret.CreatedAt, &ret.Version)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return e.ErrRecordNotFound
			}
			return err
		}

		r.setFieldValue(model, "CreatedAt", ret.CreatedAt)
		r.setFieldValue(model, "Version", ret.Version)
	} else {
		var createdAt time.Time
		var version int

		pkPtr := reflect.New(reflect.TypeOf(r.getFieldValue(model, pk.fieldName))).Interface()

		err = tx.QueryRowContext(ctx, queryStr, args...).Scan(pkPtr, &createdAt, &version)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return e.ErrRecordNotFound
			}
			return err
		}

		r.setFieldValue(model, pk.fieldName, reflect.ValueOf(pkPtr).Elem().Interface())
		r.setFieldValue(model, "CreatedAt", createdAt)
		r.setFieldValue(model, "Version", version)
	}

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

	pk, err := r.getPkInfo(model)
	if err != nil {
		return fmt.Errorf("get pk info: %w", err)
	}
	values[pk.columnName] = id

	maps.Copy(values, cfg.extraParams)

	if !cfg.skipAudit {
		user := contexts.ContextGetUser(ctx)
		maps.Copy(values, map[string]any{
			"updated_at": time.Now(),
			"updated_by": user.GetID(),
		})
	}

	extraWhere := cfg.extraWhere
	if extraWhere != "" {
		extraWhere += " AND version = :version"
	} else {
		extraWhere = "version = :version"
	}
	values["version"] = r.getFieldValue(model, "Version")

	query := BuildUpdateQuery(r.table, params, pk.columnName, extraWhere, []string{"version"}, cfg)
	filteredValues := filterValuesForQuery(values, query)
	queryStr, args := NamedQuery(query, filteredValues)
	r.logger.PrintInfo(utils.MinifySQL(queryStr), nil)

	var newVersion int
	err = tx.QueryRowContext(ctx, queryStr, args...).Scan(&newVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return e.ErrEditConflict
		}
		return err
	}

	r.setFieldValue(model, "Version", newVersion)
	return nil
}
