package utils

import (
	"database/sql"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"reflect"
	"strings"
)

type Envelope map[string]any

func MinifySQL(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func WriteJSON(w http.ResponseWriter, status int, data any, headers http.Header) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	js = append(js, '\n')
	maps.Copy(w.Header(), headers)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)
	return nil
}

func GetTypeName(v any) string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return strings.ToLower(t.Name())
}

func RunInTx(
	db *sql.DB,
	fn func(tx *sql.Tx) error,
) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	fnErr := fn(tx)
	if fnErr == nil {
		return tx.Commit()
	}

	if rbErr := tx.Rollback(); rbErr != nil {
		return errors.Join(fnErr, rbErr)
	}

	return fnErr
}
