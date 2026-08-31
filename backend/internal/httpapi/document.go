package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/n8node/aiapp/internal/authn"
	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/service"
)

type DocumentHandler struct {
	docs *service.DocumentService
}

func NewDocumentHandler(docs *service.DocumentService) *DocumentHandler {
	return &DocumentHandler{docs: docs}
}

func (h *DocumentHandler) actor(r *http.Request) (userID, sessionID string, ok bool) {
	userID, ok = authn.UserID(r.Context())
	if !ok {
		return "", "", false
	}
	sessionID, ok = authn.SessionID(r.Context())
	return userID, sessionID, ok
}

func (h *DocumentHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.docs == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	userID, sessionID, ok := h.actor(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	items, err := h.docs.List(r.Context(), userID, sessionID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"documents": items}})
}

func (h *DocumentHandler) Get(w http.ResponseWriter, r *http.Request) {
	if h.docs == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	userID, sessionID, ok := h.actor(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	doc, err := h.docs.Get(r.Context(), userID, sessionID, chi.URLParam(r, "documentID"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": doc})
}

func (h *DocumentHandler) FromFile(w http.ResponseWriter, r *http.Request) {
	if h.docs == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	userID, sessionID, ok := h.actor(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var req model.DocumentFromFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	out, err := h.docs.FromFile(r.Context(), userID, sessionID, req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	code := http.StatusOK
	if out.Created || out.JobID != nil {
		code = http.StatusAccepted
	}
	writeJSON(w, code, map[string]any{"data": out})
}

func (h *DocumentHandler) Review(w http.ResponseWriter, r *http.Request) {
	if h.docs == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	userID, sessionID, ok := h.actor(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var req model.DocumentReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	doc, err := h.docs.Review(r.Context(), userID, sessionID, chi.URLParam(r, "documentID"), chi.URLParam(r, "versionID"), req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": doc})
}

func (h *DocumentHandler) Retry(w http.ResponseWriter, r *http.Request) {
	if h.docs == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	userID, sessionID, ok := h.actor(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	out, err := h.docs.Retry(r.Context(), userID, sessionID, chi.URLParam(r, "documentID"), chi.URLParam(r, "versionID"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"data": out})
}
