package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/n8node/aiapp/internal/authn"
	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/service"
)

type StorageHandler struct {
	storage *service.StorageSettingsService
}

func NewStorageHandler(storage *service.StorageSettingsService) *StorageHandler {
	return &StorageHandler{storage: storage}
}

func (h *StorageHandler) Get(w http.ResponseWriter, r *http.Request) {
	if h.storage == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	view, err := h.storage.GetAdminView(r.Context())
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": view})
}

func (h *StorageHandler) Save(w http.ResponseWriter, r *http.Request) {
	if h.storage == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	actor, _ := authn.UserID(r.Context())
	var req model.StorageAdminUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	view, err := h.storage.Update(r.Context(), actor, req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": view})
}

func (h *StorageHandler) Test(w http.ResponseWriter, r *http.Request) {
	if h.storage == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	result, err := h.storage.TestConnection(r.Context())
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}
