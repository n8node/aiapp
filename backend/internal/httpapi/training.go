package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/n8node/aiapp/internal/authn"
	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/service"
)

type TrainingHandler struct {
	training *service.TrainingService
}

func NewTrainingHandler(training *service.TrainingService) *TrainingHandler {
	return &TrainingHandler{training: training}
}

func (h *TrainingHandler) actor(r *http.Request) (userID, sessionID string, ok bool) {
	userID, ok = authn.UserID(r.Context())
	if !ok {
		return "", "", false
	}
	sessionID, ok = authn.SessionID(r.Context())
	return userID, sessionID, ok
}

func (h *TrainingHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.training == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	items, total, err := h.training.List(r.Context(), userID, sessionID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"requests": items, "total": total}})
}

func (h *TrainingHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.training == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var req model.TrainingRequestCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	item, err := h.training.Create(r.Context(), userID, sessionID, req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": item})
}

func (h *TrainingHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.training == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	item, err := h.training.Get(r.Context(), userID, sessionID, chi.URLParam(r, "requestID"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": item})
}

func (h *TrainingHandler) Submit(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.training == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	item, err := h.training.Submit(r.Context(), userID, sessionID, chi.URLParam(r, "requestID"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": item})
}

func (h *TrainingHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.training == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	item, err := h.training.Cancel(r.Context(), userID, sessionID, chi.URLParam(r, "requestID"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": item})
}

func (h *TrainingHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	if h.training == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	items, total, err := h.training.AdminList(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"requests": items, "total": total}})
}

func (h *TrainingHandler) AdminApprove(w http.ResponseWriter, r *http.Request) {
	h.adminNote(w, r, h.training.Approve)
}

func (h *TrainingHandler) AdminReject(w http.ResponseWriter, r *http.Request) {
	h.adminNote(w, r, h.training.Reject)
}

func (h *TrainingHandler) adminNote(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, actorID, id, note string) (*model.TrainingRequest, error)) {
	actor, _ := authn.UserID(r.Context())
	if h.training == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	var req model.TrainingRequestAction
	_ = json.NewDecoder(r.Body).Decode(&req)
	item, err := fn(r.Context(), actor, chi.URLParam(r, "requestID"), req.Note)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": item})
}
