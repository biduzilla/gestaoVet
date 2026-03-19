package usuario

import (
	"gestaoVet/internal/core/contexts"
	"gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/handler"
	"gestaoVet/internal/core/validator"
	"gestaoVet/utils"
	"net/http"
)

type usuarioHandler struct {
	service    UsuarioService
	errHandler errors.ErrorHandler
}

type UsuarioHandler interface {
	FindByAll(w http.ResponseWriter, r *http.Request)
	FindByID(w http.ResponseWriter, r *http.Request)
	Save(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	Delete(w http.ResponseWriter, r *http.Request)
}

func NewHandler(
	service UsuarioService,
	errHandler errors.ErrorHandler,
) *usuarioHandler {
	return &usuarioHandler{
		service:    service,
		errHandler: errHandler,
	}
}

func (h *usuarioHandler) FindByAll(w http.ResponseWriter, r *http.Request) {
	var input struct {
		nome     string
		telefone string
		email    string
		cnpj     string
		filters.Filters
	}

	v := validator.New()
	input.cnpj = handler.ReadStringParam(r, "cnpj", "")
	input.telefone = handler.ReadStringParam(r, "telefone", "")
	input.nome = handler.ReadStringParam(r, "nome", "")
	input.email = handler.ReadStringParam(r, "email", "")

	input.Filters.Page = handler.ReadIntParam(r, "page", 1, v)
	input.Filters.PageSize = handler.ReadIntParam(r, "page_size", 20, v)
	input.Filters.Sort = handler.ReadStringParam(r, "sort", "id")
	input.Filters.SortSafelist = []string{"id", "nome", "-id", "-nome"}

	models, metadata, err := h.service.FindAll(
		input.nome,
		input.telefone,
		input.email,
		input.cnpj,
		input.Filters,
	)
	if err != nil {
		h.errHandler.HandlerError(w, r, err, v)
		return
	}

	dtos := make([]*UsuarioDTO, 0, len(models))
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

func (h *usuarioHandler) FindByID(w http.ResponseWriter, r *http.Request) {
	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	user := contexts.ContextGetUser(r)
	model, err := h.service.FindByID(id, user.GetCNPJ())
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

func (h *usuarioHandler) Save(w http.ResponseWriter, r *http.Request) {
	var dto UsuarioDTO
	if err := handler.ReadJSON(w, r, &dto); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	model, err := dto.toModel()
	if err != nil {
		h.errHandler.HandlerError(w, r, err, nil)
		return
	}

	if err := h.service.Save(v, model); err != nil {
		h.errHandler.HandlerError(w, r, err, v)
		return
	}

	handler.Respond(w, r, http.StatusCreated, model.toDTO(), nil, h.errHandler)
}

func (h *usuarioHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dto UsuarioDTO
	if err := handler.ReadJSON(w, r, &dto); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	user := contexts.ContextGetUser(r)
	model, err := dto.toModel()
	if err != nil {
		h.errHandler.HandlerError(w, r, err, nil)
		return
	}

	if err := h.service.Update(v, model, user.GetCNPJ(), user.GetID()); err != nil {
		h.errHandler.HandlerError(w, r, err, v)
		return
	}

	handler.Respond(w, r, http.StatusOK, model.toDTO(), nil, h.errHandler)
}

func (h *usuarioHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
