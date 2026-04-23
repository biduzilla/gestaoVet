package repository

import (
	"fmt"
	"reflect"
	"strings"
)

type FieldParam struct {
	Column string
	Value  any
	Tag    string
}

func CollectParams(model any, mode string) ([]FieldParam, error) {
	v := reflect.ValueOf(model)

	if v.Kind() == reflect.Ptr {
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
) string {
	var columns []string
	var placeholders []string

	for _, p := range params {
		columns = append(columns, p.Column)
		placeholders = append(placeholders, ":"+p.Column)
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
	idColumn string,
	extraWhere string,
	returning []string,
) string {
	var setParts []string
	for _, p := range params {
		setParts = append(
			setParts,
			fmt.Sprintf("%s = :%s", p.Column, p.Column),
		)
	}

	where := fmt.Sprintf("%s = :%s", idColumn, idColumn)
	if extraWhere != "" {
		where += " " + extraWhere
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		table,
		strings.Join(setParts, ", "),
		where,
	)

	if len(returning) > 0 {
		query += fmt.Sprintf(
			" RETURNING %s",
			strings.Join(returning, ", "),
		)
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
