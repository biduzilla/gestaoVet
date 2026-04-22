package empresa

import (
	"gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/handler"
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
	cnpj := handler.ReadStringParam(r, "cnpj", "")
	nomeFantasia := handler.ReadStringParam(r, "nomeFantasia", "")
	razaoSocial := handler.ReadStringParam(r, "razaoSocial", "")
	email := handler.ReadStringParam(r, "email", "")

	f, err := handler.GetFilters(r, []string{"cnpj", "nome_fantasia", "-nome_fantasia", "-nome_fantasia"})
	if err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	models, metadata, err := h.service.FindAll(
		r.Context(),
		cnpj,
		nomeFantasia,
		razaoSocial,
		email,
		f,
	)

	if err != nil {
		h.errHandler.HandlerError(w, r, err)
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

	model, err := h.service.FindByCnpj(r.Context(), cnpj)
	if err != nil {
		h.errHandler.HandlerError(w, r, err)
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

	model := dto.toModel()

	if err := h.service.Save(r.Context(), model); err != nil {
		h.errHandler.HandlerError(w, r, err)
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

	model := dto.toModel()

	if err := h.service.Update(r.Context(), model); err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusOK, model.toDTO(), nil, h.errHandler)
}

func (h *empresaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context()); err != nil {
		h.errHandler.HandlerError(w, r, err)
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
