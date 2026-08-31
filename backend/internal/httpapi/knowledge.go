package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/n8node/aiapp/internal/authn"
	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/service"
)

type KnowledgeHandler struct {
	kb *service.KnowledgeService
}

func NewKnowledgeHandler(kb *service.KnowledgeService) *KnowledgeHandler {
	return &KnowledgeHandler{kb: kb}
}

func (h *KnowledgeHandler) actor(r *http.Request) (userID, sessionID string, ok bool) {
	userID, ok = authn.UserID(r.Context())
	if !ok {
		return "", "", false
	}
	sessionID, ok = authn.SessionID(r.Context())
	return userID, sessionID, ok
}

func (h *KnowledgeHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.kb == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	items, err := h.kb.List(r.Context(), userID, sessionID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"knowledge_bases": items}})
}

func (h *KnowledgeHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.kb == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var req model.KnowledgeBaseCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	k, err := h.kb.Create(r.Context(), userID, sessionID, req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": k})
}

func (h *KnowledgeHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.kb == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	k, err := h.kb.Get(r.Context(), userID, sessionID, chi.URLParam(r, "kbID"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": k})
}

func (h *KnowledgeHandler) Patch(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.kb == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var req model.KnowledgeBasePatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	k, err := h.kb.Patch(r.Context(), userID, sessionID, chi.URLParam(r, "kbID"), req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": k})
}

func (h *KnowledgeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.kb == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	if err := h.kb.Delete(r.Context(), userID, sessionID, chi.URLParam(r, "kbID")); err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *KnowledgeHandler) Vectorize(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.kb == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	out, err := h.kb.Vectorize(r.Context(), userID, sessionID, chi.URLParam(r, "kbID"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"data": out})
}

func (h *KnowledgeHandler) Search(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.kb == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var req model.KnowledgeSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	hits, err := h.kb.Search(r.Context(), userID, sessionID, chi.URLParam(r, "kbID"), req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"hits": hits}})
}
