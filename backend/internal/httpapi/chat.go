package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/n8node/aiapp/internal/authn"
	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/service"
)

type ChatHandler struct {
	chat *service.ChatService
}

func NewChatHandler(chat *service.ChatService) *ChatHandler {
	return &ChatHandler{chat: chat}
}

func (h *ChatHandler) actor(r *http.Request) (userID, sessionID string, ok bool) {
	userID, ok = authn.UserID(r.Context())
	if !ok {
		return "", "", false
	}
	sessionID, ok = authn.SessionID(r.Context())
	return userID, sessionID, ok
}

func (h *ChatHandler) Catalog(w http.ResponseWriter, r *http.Request) {
	if h.chat == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	items, err := h.chat.Catalog(r.Context())
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"models": items}})
}

func (h *ChatHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.chat == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	items, err := h.chat.List(r.Context(), userID, sessionID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"threads": items}})
}

func (h *ChatHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.chat == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var req model.ChatThreadCreateRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	t, err := h.chat.Create(r.Context(), userID, sessionID, req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": t})
}

func (h *ChatHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.chat == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	t, err := h.chat.Get(r.Context(), userID, sessionID, chi.URLParam(r, "threadID"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	msgs, err := h.chat.Messages(r.Context(), userID, sessionID, t.ID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"thread": t, "messages": msgs}})
}

func (h *ChatHandler) Patch(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.chat == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var req model.ChatThreadPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	t, err := h.chat.Patch(r.Context(), userID, sessionID, chi.URLParam(r, "threadID"), req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": t})
}

func (h *ChatHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.chat == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	if err := h.chat.Delete(r.Context(), userID, sessionID, chi.URLParam(r, "threadID")); err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *ChatHandler) Send(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.chat == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var req model.ChatMessageCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	started := false
	var flusher http.Flusher
	_, err := h.chat.Send(r.Context(), userID, sessionID, chi.URLParam(r, "threadID"), req.Content, func(ev model.ChatStreamEvent) error {
		if !started {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("X-Accel-Buffering", "no")
			flusher, _ = w.(http.Flusher)
			started = true
		}
		b, merr := json.Marshal(ev)
		if merr != nil {
			return merr
		}
		if _, werr := fmt.Fprintf(w, "data: %s\n\n", b); werr != nil {
			return werr
		}
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	})
	if err != nil && !started {
		writeAuthError(w, err)
		return
	}
	if err != nil && started {
		b, _ := json.Marshal(model.ChatStreamEvent{Type: "error"})
		_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
		if flusher != nil {
			flusher.Flush()
		}
	}
}

func (h *ChatHandler) Media(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.chat == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var req model.ChatMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	msg, err := h.chat.GenerateMedia(r.Context(), userID, sessionID, chi.URLParam(r, "threadID"), req.Kind, req.Prompt)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": msg})
}
