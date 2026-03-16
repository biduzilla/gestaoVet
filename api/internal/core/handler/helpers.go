package handler

import (
	"fmt"
	"gestaoVet/internal/core/domain/errors"
	"gestaoVet/utils"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
)

func ParseIntID(
	w http.ResponseWriter,
	r *http.Request,
	errRsp errors.ErrorHandler,
) (int64, bool) {
	id, err := readIntPathVariable(r, "id")
	if err != nil {
		errRsp.BadRequestResponse(w, r, err)
		return 0, false
	}
	return id, true
}

func ParseUUID(
	w http.ResponseWriter,
	r *http.Request,
	errRsp errors.ErrorHandler,
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

func Respond(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	data any,
	headers http.Header,
	errRsp errors.ErrorHandler,
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
