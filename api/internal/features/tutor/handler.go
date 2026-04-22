package tutor

import (
	"gestaoVet/internal/core/domain/errors"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/handler"
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
	search := handler.ReadStringParam(r, "search", "")

	f, err := handler.GetFilters(r, []string{"cnpj", "nome", "-cnpj", "-nome"})
	if err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	models, metadata, err := h.service.FindAllBySearch(r.Context(), search, f)

	if err != nil {
		h.errHandler.HandlerError(w, r, err)
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

func (h *tutorHandler) Save(w http.ResponseWriter, r *http.Request) {
	var dto TutorDTO
	if err := handler.ReadJSON(w, r, &dto); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	model := dto.ToModel()

	if err := h.service.Save(r.Context(), model, nil); err != nil {
		h.errHandler.HandlerError(w, r, err)
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

	model := dto.ToModel()

	if err := h.service.Update(r.Context(), model, nil); err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusOK, model.toDTO(), nil, h.errHandler)
}

func (h *tutorHandler) DeleteByID(w http.ResponseWriter, r *http.Request) {
	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	if err := h.service.DeleteByID(r.Context(), id, nil); err != nil {
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
