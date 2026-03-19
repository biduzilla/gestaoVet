package empresa

import (
	"gestaoVet/internal/core/contexts"
	"gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/handler"
	"gestaoVet/internal/core/validator"
	"gestaoVet/utils"
	"net/http"

	"github.com/google/uuid"
)

type empresaHandler struct {
	service    EmpresaService
	errHandler errors.ErrorHandler
}

func NewHandler(
	service EmpresaService,
	errHandler errors.ErrorHandler,
) *empresaHandler {
	return &empresaHandler{
		service:    service,
		errHandler: errHandler,
	}
}

type EmpresaHandler interface {
	FindByAll(w http.ResponseWriter, r *http.Request)
	FindByCnpj(w http.ResponseWriter, r *http.Request)
	Save(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	Delete(w http.ResponseWriter, r *http.Request)
}

func (h *empresaHandler) FindByAll(w http.ResponseWriter, r *http.Request) {
	var input struct {
		cnpj,
		nomeFantasia,
		razaoSocial,
		email string
		filters.Filters
	}

	v := validator.New()
	input.cnpj = handler.ReadStringParam(r, "cnpj", "")
	input.nomeFantasia = handler.ReadStringParam(r, "nomeFantasia", "")
	input.razaoSocial = handler.ReadStringParam(r, "razaoSocial", "")
	input.email = handler.ReadStringParam(r, "email", "")

	input.Filters.Page = handler.ReadIntParam(r, "page", 1, v)
	input.Filters.PageSize = handler.ReadIntParam(r, "page_size", 20, v)
	input.Filters.Sort = handler.ReadStringParam(r, "sort", "id")
	input.Filters.SortSafelist = []string{"id", "nome", "-id", "-nome"}

	models, metadata, err := h.service.FindAll(
		input.cnpj,
		input.nomeFantasia,
		input.razaoSocial,
		input.email,
		input.Filters,
	)
	if err != nil {
		h.errHandler.HandlerError(w, r, err, v)
		return
	}

	dtos := make([]*EmpresaDTO, 0, len(models))
	for _, m := range models {
		dtos = append(dtos, m.toDTO())
	}

	handler.Respond(
		w,
		r,
		http.StatusOK,
		utils.Envelope{
			utils.GetTypeName(dtos[0]): dtos,
			"metadata":                 metadata,
		},
		nil,
		h.errHandler,
	)
}

func (h *empresaHandler) FindByCnpj(w http.ResponseWriter, r *http.Request) {
	cnpj, ok := handler.ParseStringField(w, r, h.errHandler, "cnpj")
	if !ok {
		return
	}

	model, err := h.service.FindByCnpj(cnpj)
	if err != nil {
		h.errHandler.HandlerError(w, r, err, nil)
		return
	}

	handler.Respond(
		w, r,
		http.StatusOK,
		model.toDTO(),
		nil,
		h.errHandler,
	)
}

func (h *empresaHandler) Save(w http.ResponseWriter, r *http.Request) {
	var dto EmpresaDTO
	if err := handler.ReadJSON(w, r, &dto); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	model := dto.toModel()

	if err := h.service.Save(model, v, uuid.Nil); err != nil {
		h.errHandler.HandlerError(w, r, err, v)
		return
	}

	handler.Respond(w, r, http.StatusCreated, model.toDTO(), nil, h.errHandler)
}

func (h *empresaHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dto EmpresaDTO
	if err := handler.ReadJSON(w, r, &dto); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	user := contexts.ContextGetUser(r)
	model := dto.toModel()

	if err := h.service.Save(model, v, user.GetID()); err != nil {
		h.errHandler.HandlerError(w, r, err, v)
		return
	}

	handler.Respond(w, r, http.StatusOK, model.toDTO(), nil, h.errHandler)
}

func (h *empresaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cnpj, ok := handler.ParseStringField(w, r, h.errHandler, "cnpj")
	if !ok {
		return
	}

	user := contexts.ContextGetUser(r)
	if err := h.service.Delete(cnpj, user.GetID()); err != nil {
		h.errHandler.HandlerError(w, r, err, nil)
		return
	}

	handler.Respond(
		w,
		r,
		http.StatusNoContent,
		nil,
		nil,
		h.errHandler,
	)
}
