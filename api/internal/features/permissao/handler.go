package permissao

import (
	"gestaoVet/internal/core/contexts"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/handler"
	"gestaoVet/internal/core/validator"
	"gestaoVet/utils"
	"net/http"
)

type cargoHandler struct {
	service    CargoService
	errHandler e.ErrorHandler
}

func NewCargoHandler(service CargoService, errHandler e.ErrorHandler) CargoHandler {
	return &cargoHandler{
		service:    service,
		errHandler: errHandler,
	}
}

type CargoHandler interface {
	FindAll(w http.ResponseWriter, r *http.Request)
	FindByID(w http.ResponseWriter, r *http.Request)
	Save(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	Delete(w http.ResponseWriter, r *http.Request)
}

func (h *cargoHandler) FindAll(w http.ResponseWriter, r *http.Request) {
	var input struct {
		nome string
		filters.Filters
	}

	v := validator.New()
	input.nome = handler.ReadStringParam(r, "nome", "")
	input.Filters.Page = handler.ReadIntParam(r, "page", 1, v)
	input.Filters.PageSize = handler.ReadIntParam(r, "page_size", 20, v)
	input.Filters.Sort = handler.ReadStringParam(r, "sort", "id")
	input.Filters.SortSafelist = []string{"id", "nome", "-id", "-nome"}

	user := contexts.ContextGetUser(r)
	cargos, metadata, err := h.service.FindAll(
		input.nome,
		input.Filters,
		user.GetCNPJ(),
	)
	if err != nil {
		h.errHandler.HandlerError(w, r, err, v)
		return
	}

	dtos := make([]*CargoDTO, 0, len(cargos))
	for _, c := range cargos {
		dtos = append(dtos, c.toDTO())
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

func (h *cargoHandler) FindByID(w http.ResponseWriter, r *http.Request) {
	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	user := contexts.ContextGetUser(r)
	cargo, err := h.service.FindByID(id, user.GetCNPJ())
	if err != nil {
		h.errHandler.HandlerError(w, r, err, nil)
		return
	}

	handler.Respond(
		w,
		r,
		http.StatusOK,
		cargo.toDTO(),
		nil,
		h.errHandler,
	)
}

func (h *cargoHandler) Save(w http.ResponseWriter, r *http.Request) {
	var dto CargoDTO
	if err := handler.ReadJSON(w, r, &dto); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	model := dto.toModel()
	user := contexts.ContextGetUser(r)
	v := validator.New()

	if err := h.service.Insert(v, model, user.GetID()); err != nil {
		h.errHandler.HandlerError(w, r, err, v)
		return
	}

	handler.Respond(w, r, http.StatusCreated, model.toDTO(), nil, h.errHandler)
}

func (h *cargoHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dto CargoDTO
	if err := handler.ReadJSON(w, r, &dto); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	model := dto.toModel()
	user := contexts.ContextGetUser(r)
	v := validator.New()

	if err := h.service.Update(v, model, user.GetCNPJ(), user.GetID()); err != nil {
		h.errHandler.HandlerError(w, r, err, v)
		return
	}

	handler.Respond(w, r, http.StatusOK, model.toDTO(), nil, h.errHandler)
}

func (h *cargoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	user := contexts.ContextGetUser(r)
	if err := h.service.Delete(id, user.GetID(), user.GetCNPJ()); err != nil {
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
