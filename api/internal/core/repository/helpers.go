package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"reflect"
	"slices"
	"strings"

	"github.com/lib/pq"
)

type FieldParam struct {
	Column string
	Value  any
	Tag    string
}

func CollectParams(model any, mode string, ignoreFields ...string) ([]FieldParam, error) {
	v := reflect.ValueOf(model)

	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, fmt.Errorf("model cannot be nil")
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", v.Kind())
	}

	t := v.Type()
	var params []FieldParam

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldVal := v.Field(i)

		dbTag := field.Tag.Get("db")
		repoTag := field.Tag.Get("repo")

		shouldIgnore := slices.Contains(ignoreFields, dbTag)

		if shouldIgnore {
			continue
		}

		if field.Anonymous && fieldVal.Kind() == reflect.Struct {
			nestedParams, err := collectNestedParams(fieldVal, mode)
			if err == nil && len(nestedParams) > 0 {
				params = append(params, nestedParams...)
			}
			continue
		}

		if fieldVal.Kind() == reflect.Struct && dbTag == "-" {
			nestedParams, err := collectNestedParams(fieldVal, mode)
			if err == nil && len(nestedParams) > 0 {
				params = append(params, nestedParams...)
			}
			continue
		}

		if dbTag == "" || dbTag == "-" {
			continue
		}

		if shouldSkipField(repoTag, mode) {
			continue
		}

		value := fieldVal.Interface()
		params = append(params, FieldParam{
			Column: dbTag,
			Value:  value,
			Tag:    repoTag,
		})
	}

	return params, nil
}

func collectNestedParams(v reflect.Value, mode string) ([]FieldParam, error) {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct")
	}

	t := v.Type()
	var params []FieldParam

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldVal := v.Field(i)
		dbTag := field.Tag.Get("db")
		repoTag := field.Tag.Get("repo")

		if field.Anonymous && fieldVal.Kind() == reflect.Struct {
			nested, err := collectNestedParams(fieldVal, mode)
			if err == nil && len(nested) > 0 {
				params = append(params, nested...)
			}
			continue
		}

		if dbTag == "" || dbTag == "-" {
			continue
		}
		if shouldSkipField(repoTag, mode) {
			continue
		}

		params = append(params, FieldParam{
			Column: dbTag,
			Value:  fieldVal.Interface(),
			Tag:    repoTag,
		})
	}
	return params, nil
}

func shouldSkipField(repoTag, mode string) bool {
	if repoTag == "" {
		return false
	}

	tags := strings.Split(repoTag, ",")
	hasInsert := false
	hasUpdate := false
	hasNoInsert := false
	hasNoUpdate := false
	isAuto := false
	isVersion := false

	for _, tag := range tags {
		switch strings.TrimSpace(tag) {
		case "insert":
			hasInsert = true
		case "update":
			hasUpdate = true
		case "noinsert":
			hasNoInsert = true
		case "noupdate":
			hasNoUpdate = true
		case "auto":
			isAuto = true
		case "version":
			isVersion = true
		}
	}

	if isAuto || isVersion {
		return true
	}

	if mode == "insert" {
		if hasInsert || hasNoInsert {
			return hasNoInsert || !hasInsert
		}
		return false
	}

	if mode == "update" {
		if hasUpdate || hasNoUpdate {
			return hasNoUpdate || !hasUpdate
		}
		return false
	}

	return false
}

func BuildInsertQuery(
	table string,
	params []FieldParam,
	returning []string,
	cfg *mutationConfig,
) string {
	var columns []string
	var placeholders []string

	for _, p := range params {
		columns = append(columns, p.Column)
		placeholders = append(placeholders, ":"+p.Column)
	}

	if !cfg.skipAudit {
		columns = append(columns, "created_by", "created_at")
		placeholders = append(placeholders, ":created_by", ":created_at")
	}

	for _, f := range cfg.extraFields {
		columns = append(columns, f)
		placeholders = append(placeholders, ":"+f)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	if len(returning) > 0 {
		query += fmt.Sprintf(
			" RETURNING %s",
			strings.Join(returning, ", "),
		)
	}

	return query
}

func BuildUpdateQuery(
	table string,
	params []FieldParam,
	pkColumn string,
	extraWhere string,
	returning []string,
	cfg *mutationConfig,
) string {
	var setParts []string
	for _, p := range params {
		if strings.Contains(p.Tag, "pk") {
			continue
		}
		setParts = append(setParts, fmt.Sprintf("%s = :%s", p.Column, p.Column))
	}

	where := fmt.Sprintf("%s = :%s AND deleted = false", pkColumn, pkColumn)
	if extraWhere != "" {
		where += " and " + extraWhere
	}

	if !cfg.skipAudit {
		setParts = append(setParts, "updated_at = :updated_at")
		setParts = append(setParts, "updated_by = :updated_by")
		setParts = append(setParts, "version = version + 1")
	}

	for _, f := range cfg.extraFields {
		setParts = append(setParts, fmt.Sprintf("%s = :%s", f, f))
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		table,
		strings.Join(setParts, ", "),
		where,
	)

	if len(returning) > 0 {
		query += fmt.Sprintf(" RETURNING %s", strings.Join(returning, ", "))
	}

	return query
}

func ParamsToMap(params []FieldParam) map[string]any {
	m := make(map[string]any, len(params))
	for _, p := range params {
		m[p.Column] = p.Value
	}
	return m
}

func NamedQuery(query string, params map[string]any) (string, []any) {
	args := []any{}
	i := 1

	for key, value := range params {
		placeholder := fmt.Sprintf("$%d", i)
		paramName := ":" + key
		query = strings.ReplaceAll(query, paramName, placeholder)
		args = append(args, value)
		i++
	}

	return query, args
}

func ListQuery[T any](
	ctx context.Context,
	db *sql.DB,
	query string,
	args []any,
	factory FactoryFunc[T],
) ([]*T, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	models := []*T{}

	for rows.Next() {
		model := factory()

		fields, err := collectFields(model)
		if err != nil {
			return nil, err
		}

		if err := rows.Scan(fields...); err != nil {
			return nil, err
		}

		models = append(models, model)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return models, nil
}

func PaginatedQuery[T any](
	ctx context.Context,
	db *sql.DB,
	query string,
	args []any,
	f filters.Filters,
	factory FactoryFunc[T],
) ([]*T, filters.Metadata, error) {
	rows, err := db.QueryContext(ctx, query, args...)

	if err != nil {
		return nil, filters.Metadata{}, err
	}

	defer rows.Close()

	totalRecords := 0
	models := []*T{}

	for rows.Next() {
		var total int

		model := factory()

		fields, err := collectFields(model)
		if err != nil {
			return nil, filters.Metadata{}, err
		}

		scanArgs := append([]any{&total}, fields...)

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, filters.Metadata{}, err
		}

		totalRecords = total
		models = append(models, model)
	}

	if err = rows.Err(); err != nil {
		return nil, filters.Metadata{}, err
	}

	metaData := filters.CalculateMetadata(totalRecords, f.Page, f.PageSize)
	return models, metaData, nil
}

func GetByQuery[T any](
	ctx context.Context,
	db *sql.DB,
	query string,
	args []any,
) (*T, error) {
	var model T
	row := db.QueryRowContext(ctx, query, args...)
	err := ScanStruct(row, &model)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, e.ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &model, nil
}

type FactoryFunc[T any] func() *T

func ScanStruct(row *sql.Row, dest any) error {
	fields, err := collectFields(dest)
	if err != nil {
		return err
	}

	err = row.Scan(fields...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return e.ErrRecordNotFound
		}
		return err
	}

	return nil
}

func SelectColumns(cfg *queryConfig, model any, tableAlias string) string {
	cols := []string{}
	collectColumns(reflect.TypeOf(model), tableAlias, &cols)

	for _, join := range cfg.joins {
		collectColumns(reflect.TypeOf(join.Model), join.Alias, &cols)
	}

	return strings.Join(cols, ", ")
}

func collectColumns(t reflect.Type, alias string, cols *[]string) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return
	}

	for field := range t.Fields() {
		field := field

		tag := field.Tag.Get("db")

		// if tag == "-" {
		// 	continue
		// }

		if tag != "" && tag != "-" {
			*cols = append(*cols, fmt.Sprintf("%s.%s as %s_%s", alias, tag, alias, tag))
		}

		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			collectColumns(field.Type, alias, cols)
			continue
		}

		// if field.Type.Kind() == reflect.Pointer &&
		// 	field.Type.Elem().Kind() == reflect.Struct {

		// 	collectColumns(field.Type.Elem(), alias, cols)
		// 	continue
		// }

		if field.Type.Kind() == reflect.Struct {
			collectColumns(field.Type, alias, cols)
		}
	}
}

func collectFields(dest any) ([]any, error) {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Pointer {
		return nil, e.ErrScanModel
	}

	v = v.Elem()
	t := v.Type()

	var fields []any

	for i := 0; i < t.NumField(); i++ {
		fieldVal := v.Field(i)
		fieldType := t.Field(i)
		tag := fieldType.Tag.Get("db")

		if tag != "" && tag != "-" {
			if fieldVal.Kind() == reflect.Slice {
				elemKind := fieldVal.Type().Elem().Kind()

				switch elemKind {
				case reflect.Uint8:
					fields = append(fields, fieldVal.Addr().Interface())
					continue
				case reflect.String, reflect.Int, reflect.Int32, reflect.Int64:
					fields = append(fields, pq.Array(fieldVal.Addr().Interface()))
					continue
				default:
					return nil, e.ErrUnsupportedTypeScanModel
				}
			} else {
				fields = append(fields, fieldVal.Addr().Interface())
				continue
			}
		}

		if fieldType.Anonymous && fieldVal.Kind() == reflect.Struct {
			subFields, err := collectFields(fieldVal.Addr().Interface())
			if err != nil {
				return nil, err
			}
			fields = append(fields, subFields...)
			continue
		}

		if fieldVal.Kind() == reflect.Pointer &&
			fieldVal.Type().Elem().Kind() == reflect.Struct {

			if fieldVal.IsNil() {
				fieldVal.Set(reflect.New(fieldVal.Type().Elem()))
			}

			subFields, err := collectFields(fieldVal.Interface())
			if err != nil {
				return nil, err
			}
			fields = append(fields, subFields...)
			continue
		}

		if fieldVal.Kind() == reflect.Struct {
			subFields, err := collectFields(fieldVal.Addr().Interface())
			if err != nil {
				return nil, err
			}
			fields = append(fields, subFields...)
		}
	}

	return fields, nil
}

func BuildFilterQuery(alias string, search ...string) string {
	if len(search) == 0 {
		return ""
	}

	var fields []string
	for _, s := range search {
		fields = append(fields, fmt.Sprintf("coalesce(%s.%s, '')", alias, s))
	}

	return fmt.Sprintf(`((:search is null or :search = '')
	or to_tsvector('simple', %s) @@ plainto_tsquery('simple', :search))`,
		strings.Join(fields, " || ' ' || "))
}
