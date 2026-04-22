package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"strings"
)

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
