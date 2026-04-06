package empresa

import (
	"gestaoVet/internal/core/contexts"
	"gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/handler"
	"gestaoVet/internal/core/validator"
	"gestaoVet/utils"
	"net/http"
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
	input.Filters.Sort = handler.ReadStringParam(r, "sort", "cnpj")
	input.Filters.SortSafelist = []string{"cnpj", "nome_fantasia", "-nome_fantasia", "-nome_fantasia"}

	if filters.ValidateFilters(v, input.Filters); !v.Valid() {
		h.errHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

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
			"content":  dtos,
			"metadata": metadata,
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

	if err := h.service.Save(model, v); err != nil {
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

	if err := h.service.Update(model, v, user.GetID(), user.GetCNPJ()); err != nil {
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
