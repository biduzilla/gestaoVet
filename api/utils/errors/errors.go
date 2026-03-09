package errors

import (
	"errors"
	"fmt"
	"gestaoVet/internal/jsonlog"
	"gestaoVet/utils"
	"gestaoVet/utils/validator"
	"net/http"
	"strings"
)

type errorHandler struct {
	logger jsonlog.Logger
}

func NewErrorHandler(logger jsonlog.Logger) *errorHandler {
	return &errorHandler{
		logger: logger,
	}
}

type ErrorHandlerInterface interface {
	NotPermittedResponse(w http.ResponseWriter, r *http.Request)
	AuthenticationRequiredResponse(w http.ResponseWriter, r *http.Request)
	InactiveAccountResponse(w http.ResponseWriter, r *http.Request)
	InvalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request)
	InvalidCredentialsResponse(w http.ResponseWriter, r *http.Request)
	InvalidRoleResponse(w http.ResponseWriter, r *http.Request)
	RateLimitExceededResponse(w http.ResponseWriter, r *http.Request)
	ServerErrorResponse(w http.ResponseWriter, r *http.Request, err error)
	NotFoundResponse(w http.ResponseWriter, r *http.Request)
	MethodNotAllowedResponse(w http.ResponseWriter, r *http.Request)
	BadRequestResponse(w http.ResponseWriter, r *http.Request, err error)
	FailedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string)
	EditConflictResponse(w http.ResponseWriter, r *http.Request)
	HandlerError(w http.ResponseWriter, r *http.Request, err error, v *validator.Validator)
}

var (
	ErrRecordNotFound           = errors.New("record not found")
	ErrEditConflict             = errors.New("edit conflict")
	ErrInvalidData              = errors.New("invalid data")
	ErrInvalidCredentials       = errors.New("invalid authentication credentials")
	ErrInactiveAccount          = errors.New("your user account must be activated to access this resource")
	ErrStartDateAfterEndDate    = errors.New("start date must be before end date")
	ErrInvalidRole              = errors.New("invalid role")
	ErrScanModel                = errors.New("dest must be a pointer")
	ErrUnsupportedTypeScanModel = errors.New("unsupported slice type for db scan")
)

func (e *errorHandler) HandlerError(w http.ResponseWriter, r *http.Request, err error, v *validator.Validator) {

	switch {
	case errors.Is(err, ErrInvalidData):
		e.FailedValidationResponse(w, r, v.Errors)

	case errors.Is(err, ErrRecordNotFound):
		e.NotFoundResponse(w, r)

	case errors.Is(err, ErrEditConflict):
		e.EditConflictResponse(w, r)

	case errors.Is(err, ErrInactiveAccount):
		e.InactiveAccountResponse(w, r)

	case errors.Is(err, ErrInvalidRole):
		e.InvalidRoleResponse(w, r)

	case errors.Is(err, ErrInvalidCredentials):
		e.InvalidCredentialsResponse(w, r)

	case len(strings.Split(err.Error(), "->")) > 1:
		parts := strings.Split(err.Error(), "->")
		for i := 0; i+1 < len(parts); i += 2 {
			e.FailedValidationResponse(w, r, map[string]string{
				parts[i]: parts[i+1],
			})
		}

	default:
		e.ServerErrorResponse(w, r, err)
	}
}

func ValidationAlreadyExists(field string) error {
	return fmt.Errorf("%s -> a register with this %s address already exists", field, field)
}

func (e *errorHandler) NotPermittedResponse(w http.ResponseWriter, r *http.Request) {
	message := "your user account doesn't have the necessary permissions to access this resource"
	e.errorHandler(w, r, http.StatusForbidden, message)
}

func (e *errorHandler) AuthenticationRequiredResponse(w http.ResponseWriter, r *http.Request) {
	message := "you must be authenticated to access this resource"
	e.errorHandler(w, r, http.StatusUnauthorized, message)
}
func (e *errorHandler) InactiveAccountResponse(w http.ResponseWriter, r *http.Request) {
	message := "your user account must be activated to access this resource"
	e.errorHandler(w, r, http.StatusForbidden, message)
}

func (e *errorHandler) InvalidRoleResponse(w http.ResponseWriter, r *http.Request) {
	message := "Your user account does not have access to this feature."
	e.errorHandler(w, r, http.StatusForbidden, message)
}

func (e *errorHandler) InvalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	message := "invalid or missing authentication token"
	e.errorHandler(w, r, http.StatusUnauthorized, message)
}

func (e *errorHandler) InvalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	message := "invalid authentication credentials"
	e.errorHandler(w, r, http.StatusUnauthorized, message)
}

func (e *errorHandler) RateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	message := "rate limit exceed"
	e.errorHandler(w, r, http.StatusTooManyRequests, message)
}

func (e *errorHandler) ServerErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	e.logError(r, err)
	message := "the server encountered a problem and could not process your request"
	e.errorHandler(w, r, http.StatusInternalServerError, message)
}

func (e *errorHandler) NotFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	e.errorHandler(w, r, http.StatusNotFound, message)
}

func (e *errorHandler) MethodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	e.errorHandler(w, r, http.StatusMethodNotAllowed, message)
}

func (e *errorHandler) BadRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	e.errorHandler(w, r, http.StatusBadRequest, err.Error())
}

func (e *errorHandler) FailedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	e.errorHandler(w, r, http.StatusUnprocessableEntity, errors)
}

func (e *errorHandler) EditConflictResponse(w http.ResponseWriter, r *http.Request) {
	message := "unable to update the record due to an edit conflict, please try again"
	e.errorHandler(w, r, http.StatusConflict, message)
}

func (e *errorHandler) errorHandler(w http.ResponseWriter, r *http.Request, status int, message any) {
	env := utils.Envelope{"error": message}
	err := utils.WriteJSON(w, status, env, nil)
	if err != nil {
		e.logError(r, err)
		w.WriteHeader(500)
	}
}

func (e *errorHandler) logError(r *http.Request, err error) {
	e.logger.PrintError(err, map[string]string{
		"request_method": r.Method,
		"request_url":    r.URL.String(),
	})
}
