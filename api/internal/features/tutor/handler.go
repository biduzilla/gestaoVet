package tutor

import (
	"gestaoVet/internal/core/contexts"
	"gestaoVet/internal/core/domain/errors"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/handler"
	"gestaoVet/internal/core/validator"
	"gestaoVet/utils"
	"net/http"
)

type tutorHandler struct {
	service    TutorService
	errHandler errors.ErrorHandler
}

type TutorHandler interface {
	FindByAll(w http.ResponseWriter, r *http.Request)
	FindByID(w http.ResponseWriter, r *http.Request)
	Save(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	DeleteByID(w http.ResponseWriter, r *http.Request)
}

func NewHandler(
	service TutorService,
	errHandler e.ErrorHandler,
) TutorHandler {
	return &tutorHandler{
		service:    service,
		errHandler: errHandler,
	}
}

func (h *tutorHandler) FindByAll(w http.ResponseWriter, r *http.Request) {
	v := validator.New()
	search := handler.ReadStringParam(r, "search", "")

	f, err := handler.GetFilters(r, v, []string{"cnpj", "nome", "-cnpj", "-nome"})
	if err != nil {
		h.errHandler.HandlerError(w, r, err, v)
		return
	}

	user := contexts.ContextGetUser(r)

	models, metadata, err := h.service.FindAllBySearch(
		search,
		user.GetCNPJ(),
		f,
	)

	if err != nil {
		h.errHandler.HandlerError(w, r, err, v)
		return
	}

	dtos := make([]*TutorDTO, 0, len(models))
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

func (h *tutorHandler) FindByID(w http.ResponseWriter, r *http.Request) {
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

func (h *tutorHandler) Save(w http.ResponseWriter, r *http.Request) {
	var dto TutorDTO
	if err := handler.ReadJSON(w, r, &dto); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	user := contexts.ContextGetUser(r)
	model, err := dto.toModel(v)
	if err != nil {
		h.errHandler.HandlerError(w, r, err, v)
		return
	}

	if err := h.service.Save(nil, model, v, user.GetCNPJ()); err != nil {
		h.errHandler.HandlerError(w, r, err, v)
		return
	}

	handler.Respond(w, r, http.StatusCreated, model.toDTO(), nil, h.errHandler)
}

func (h *tutorHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dto TutorDTO
	if err := handler.ReadJSON(w, r, &dto); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	user := contexts.ContextGetUser(r)
	model, err := dto.toModel(v)
	if err != nil {
		h.errHandler.HandlerError(w, r, err, v)
		return
	}

	if err := h.service.Update(nil, model, v, user.GetCNPJ(), user.GetID()); err != nil {
		h.errHandler.HandlerError(w, r, err, v)
		return
	}

	handler.Respond(w, r, http.StatusOK, model.toDTO(), nil, h.errHandler)
}

func (h *tutorHandler) DeleteByID(w http.ResponseWriter, r *http.Request) {
	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	user := contexts.ContextGetUser(r)
	if err := h.service.DeleteByID(nil, id, user.GetID(), user.GetCNPJ()); err != nil {
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
