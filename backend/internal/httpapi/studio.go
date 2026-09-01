package httpapi

import (
	"net/http"

	"github.com/n8node/aiapp/internal/authn"
	"github.com/n8node/aiapp/internal/service"
)

type StudioHandler struct {
	studio *service.StudioAuthService
}

func NewStudioHandler(studio *service.StudioAuthService) *StudioHandler {
	return &StudioHandler{studio: studio}
}

func (h *StudioHandler) Session(w http.ResponseWriter, r *http.Request) {
	userID, ok := authn.UserID(r.Context())
	if !ok || h.studio == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	sess, err := h.studio.Session(r.Context(), userID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": sess})
}
