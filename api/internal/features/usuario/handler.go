package usuario

import (
	"gestaoVet/internal/core/domain/errors"
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
	UpdateSenha(w http.ResponseWriter, r *http.Request)
	UpdateRoles(w http.ResponseWriter, r *http.Request)
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
	cnpj := handler.ReadStringParam(r, "cnpj", "")
	telefone := handler.ReadStringParam(r, "telefone", "")
	nome := handler.ReadStringParam(r, "nome", "")
	email := handler.ReadStringParam(r, "email", "")

	f, err := handler.GetFilters(r, []string{"id", "nome", "-id", "-nome"})
	if err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	models, metadata, err := h.service.FindAll(
		r.Context(),
		nome,
		telefone,
		email,
		cnpj,
		f,
	)

	if err != nil {
		h.errHandler.HandlerError(w, r, err)
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
			"content":  dtos,
			"metadata": metadata,
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

	model, err := h.service.FindByID(r.Context(), id)
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

func (h *usuarioHandler) Save(w http.ResponseWriter, r *http.Request) {
	var dto UsuarioDTO
	if err := handler.ReadJSON(w, r, &dto); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	model, err := dto.toModel()
	if err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	if err := h.service.Save(r.Context(), model, nil); err != nil {
		h.errHandler.HandlerError(w, r, err)
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

	model, err := dto.toModel()
	if err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	if err := h.service.Update(r.Context(), model); err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusOK, model.toDTO(), nil, h.errHandler)
}

func (h *usuarioHandler) UpdateSenha(w http.ResponseWriter, r *http.Request) {
	var dto UsuarioDTO
	if err := handler.ReadJSON(w, r, &dto); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	v.Check(dto.ID != nil, "id", "id must be provide")
	v.Check(dto.Senha != nil, "senha", "senha must be provide")
	v.Check(*dto.Senha != "", "senha", "senha must be provide")

	if !v.Valid() {
		h.errHandler.FailedValidationResponse(w, r, v.Errors)
		return
	}

	if err := h.service.UpdateSenha(r.Context(), *dto.ID, *dto.Senha); err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusNoContent, nil, nil, h.errHandler)
}

func (h *usuarioHandler) UpdateRoles(w http.ResponseWriter, r *http.Request) {
	var dto RolesDTO

	if err := handler.ReadJSON(w, r, &dto); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	if err := h.service.UpdateRoles(r.Context(), dto.ID, dto.Roles); err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusOK, nil, nil, h.errHandler)
}

func (h *usuarioHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
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
