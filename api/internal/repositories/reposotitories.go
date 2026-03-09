package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gestaoVet/internal/models/filters"
	e "gestaoVet/utils/errors"
	"reflect"
	"strings"
	"time"

	"github.com/lib/pq"
)

type repositories struct {
}

type FactoryFunc[T any] func() *T

func scanStruct(row *sql.Row, dest any) error {
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

func namedQuery(query string, params map[string]any) (string, []any) {
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

func listQuery[T any](
	db *sql.DB,
	query string,
	args []any,
	factory FactoryFunc[T],
) ([]*T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

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

func paginatedQuery[T any](
	db *sql.DB,
	query string,
	args []any,
	f filters.Filters,
	factory FactoryFunc[T],
) ([]*T, filters.Metadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

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

func getByQuery[T any](
	db *sql.DB,
	query string,
	args []any,
) (*T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var model T
	row := db.QueryRowContext(ctx, query, args...)
	err := scanStruct(row, &model)

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

func selectColumns(model any, tableAlias string) string {
	cols := []string{}
	collectColumns(reflect.TypeOf(model), tableAlias, &cols)
	return strings.Join(cols, ", ")
}

func collectColumns(t reflect.Type, alias string, cols *[]string) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		tag := field.Tag.Get("db")

		// if tag == "-" {
		// 	continue
		// }

		if tag != "" && tag != "-" {
			*cols = append(*cols, fmt.Sprintf("%s.%s", alias, tag))
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
				case reflect.String, reflect.Int, reflect.Int64:
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
