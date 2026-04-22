package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/validator"
	"gestaoVet/utils"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
)

func ParseIntID(
	w http.ResponseWriter,
	r *http.Request,
	errRsp e.ErrorHandler,
) (int64, bool) {
	id, err := readIntPathVariable(r, "id")
	if err != nil {
		errRsp.BadRequestResponse(w, r, err)
		return 0, false
	}
	return id, true
}

func GetFilters(r *http.Request, sortSafelist []string) (filters.Filters, error) {
	v := validator.New()

	var f = filters.Filters{}
	f.Page = ReadIntParam(r, "page", 1, v)
	f.PageSize = ReadIntParam(r, "page_size", 20, v)
	f.Sort = ReadStringParam(r, "sort", "cnpj")
	f.SortSafelist = sortSafelist

	if filters.ValidateFilters(v, f); !v.Valid() {
		return filters.Filters{}, e.NewValidationError(v.Errors)
	}

	return f, nil

}

func ParseStringField(
	w http.ResponseWriter,
	r *http.Request,
	errRsp e.ErrorHandler,
	field string,
) (string, bool) {
	value, err := readStringPathVariable(r, field)
	if err != nil {
		errRsp.BadRequestResponse(w, r, err)
		return "", false
	}
	return value, true
}

func ParseUUID(
	w http.ResponseWriter,
	r *http.Request,
	errRsp e.ErrorHandler,
) (uuid.UUID, bool) {

	id, err := readStringPathVariable(r, "id")
	if err != nil {
		errRsp.BadRequestResponse(w, r, err)
		return uuid.Nil, false
	}

	uid, err := uuid.Parse(id)
	if err != nil {
		errRsp.BadRequestResponse(w, r, err)
		return uuid.Nil, false
	}

	return uid, true
}

func ReadStringParam(r *http.Request, key, defaultValue string) string {
	qs := r.URL.Query()
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}
	return s
}

func ReadDate(qs url.Values, key string, layout string) *time.Time {
	s := qs.Get(key)
	if s == "" {
		return nil
	}

	t, err := time.Parse(layout, s)
	if err != nil {
		return nil
	}
	return &t
}

func ReadIntParam(r *http.Request, key string, defaultValue int, v *validator.Validator) int {
	qs := r.URL.Query()
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}
	return i
}

func ReadJSON(
	w http.ResponseWriter,
	r *http.Request,
	dst any,
) error {
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)
		case err.Error() == "http: request body too large":
			return fmt.Errorf("body must not be larger than %d bytes", maxBytes)
		case errors.As(err, &invalidUnmarshalError):
			panic(err)
		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

func Respond(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	data any,
	headers http.Header,
	errRsp e.ErrorHandler,
) {
	err := utils.WriteJSON(w, status, data, headers)
	if err != nil {
		errRsp.ServerErrorResponse(w, r, err)
		return
	}
}

func readIntPathVariable(r *http.Request, key string) (int64, error) {
	s := chi.URLParam(r, key)

	if s == "" {
		return 0, fmt.Errorf("missing path parameter: %s", key)
	}

	value, err := strconv.ParseInt(s, 10, 64)

	if err != nil {
		return 0, fmt.Errorf("invalid %s parameter", key)
	}

	return value, nil
}

func readStringPathVariable(r *http.Request, key string) (string, error) {
	s := chi.URLParam(r, key)

	if s == "" {
		return "", fmt.Errorf("missing path parameter: %s", key)
	}

	return s, nil
}
