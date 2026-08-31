package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/n8node/aiapp/internal/authn"
	"github.com/n8node/aiapp/internal/service"
)

type BitrixHandler struct {
	bitrix     *service.BitrixService
	workspaces *service.WorkspaceService
}

func NewBitrixHandler(bitrix *service.BitrixService, workspaces *service.WorkspaceService) *BitrixHandler {
	return &BitrixHandler{bitrix: bitrix, workspaces: workspaces}
}

func (h *BitrixHandler) Status(w http.ResponseWriter, r *http.Request) {
	st, err := h.bitrix.Status(r.Context())
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": st})
}

func (h *BitrixHandler) Save(w http.ResponseWriter, r *http.Request) {
	actor, _ := authn.UserID(r.Context())
	var req struct {
		WebhookURL string `json:"webhook_url"`
		Disconnect bool   `json:"disconnect"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	var (
		st  any
		err error
	)
	if req.Disconnect {
		st, err = h.bitrix.Disconnect(r.Context(), actor)
	} else {
		st, err = h.bitrix.SaveWebhook(r.Context(), actor, req.WebhookURL)
	}
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": st})
}

func (h *BitrixHandler) Test(w http.ResponseWriter, r *http.Request) {
	if err := h.bitrix.Test(r.Context()); err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *BitrixHandler) Sync(w http.ResponseWriter, r *http.Request) {
	actor, _ := authn.UserID(r.Context())
	res, err := h.bitrix.Sync(r.Context(), actor)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	data := map[string]any{"departments": res.Departments, "users": res.Users}
	if h.workspaces != nil {
		if applied, applyErr := h.workspaces.ApplyMemberships(r.Context(), actor); applyErr == nil {
			data["memberships"] = applied
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (h *BitrixHandler) Departments(w http.ResponseWriter, r *http.Request) {
	deps, err := h.bitrix.Departments(r.Context())
	if err != nil {
		writeAuthError(w, err)
		return
	}
	if h.workspaces != nil {
		deps, err = h.workspaces.EnrichDepartments(r.Context(), deps)
		if err != nil {
			writeAuthError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"departments": deps}})
}

func (h *BitrixHandler) Users(w http.ResponseWriter, r *http.Request) {
	users, err := h.bitrix.Users(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"users": users}})
}
