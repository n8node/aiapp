package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/n8node/aiapp/internal/authn"
	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/service"
)

type ModelHandler struct {
	models *service.ModelService
}

func NewModelHandler(models *service.ModelService) *ModelHandler {
	return &ModelHandler{models: models}
}

func (h *ModelHandler) AuthGate(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (h *ModelHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.models == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	items, err := h.models.List(r.Context())
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"models": items}})
}

func (h *ModelHandler) Create(w http.ResponseWriter, r *http.Request) {
	if h.models == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	actor, _ := authn.UserID(r.Context())
	var req model.MLModelCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	m, err := h.models.Create(r.Context(), actor, req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": m})
}

func (h *ModelHandler) Action(w http.ResponseWriter, r *http.Request) {
	if h.models == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	actor, _ := authn.UserID(r.Context())
	var req model.MLModelActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	m, err := h.models.Action(r.Context(), actor, chi.URLParam(r, "modelID"), req.Action)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": m})
}

func (h *ModelHandler) Runtime(w http.ResponseWriter, r *http.Request) {
	if h.models == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{
		"studio":  h.models.StudioStatus(r.Context()),
		"gateway": h.models.GatewayStatus(r.Context()),
	}})
}
