package httpapi

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/n8node/aiapp/internal/authn"
	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/service"
)

type OutboundProxyHandler struct {
	svc *service.OutboundProxyService
}

func NewOutboundProxyHandler(svc *service.OutboundProxyService) *OutboundProxyHandler {
	return &OutboundProxyHandler{svc: svc}
}

func (h *OutboundProxyHandler) Get(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	view, err := h.svc.GetAdminView(r.Context())
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": view})
}

func (h *OutboundProxyHandler) Save(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	actor, _ := authn.UserID(r.Context())
	var cfg model.OutboundProxySettings
	if err := json.NewDecoder(io.LimitReader(r.Body, 32<<10)).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	view, err := h.svc.Update(r.Context(), actor, cfg)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": view})
}

func (h *OutboundProxyHandler) Test(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	result, err := h.svc.Test(r.Context())
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (h *OutboundProxyHandler) InternalGet(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	token := r.Header.Get("X-RigIntel-Internal")
	url, err := h.svc.InternalURL(r.Context(), token)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(url))
}
