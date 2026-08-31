package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/n8node/aiapp/internal/authn"
	"github.com/n8node/aiapp/internal/service"
)

type WorkspaceHandler struct {
	workspaces *service.WorkspaceService
}

func NewWorkspaceHandler(workspaces *service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{workspaces: workspaces}
}

func (h *WorkspaceHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.workspaces.List(r.Context())
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"workspaces": list}})
}

func (h *WorkspaceHandler) CreateFromDepartment(w http.ResponseWriter, r *http.Request) {
	actor, _ := authn.UserID(r.Context())
	var req struct {
		BitrixDepartmentID int64  `json:"bitrix_department_id"`
		IncludeDescendants *bool  `json:"include_descendants"`
		Name               string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.BitrixDepartmentID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	include := true
	if req.IncludeDescendants != nil {
		include = *req.IncludeDescendants
	}
	ws, err := h.workspaces.CreateFromDepartment(r.Context(), actor, req.BitrixDepartmentID, include, req.Name)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": ws})
}

func (h *WorkspaceHandler) LinkDepartment(w http.ResponseWriter, r *http.Request) {
	actor, _ := authn.UserID(r.Context())
	workspaceID := chi.URLParam(r, "workspaceID")
	var req struct {
		BitrixDepartmentID int64 `json:"bitrix_department_id"`
		IncludeDescendants *bool `json:"include_descendants"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.BitrixDepartmentID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	include := true
	if req.IncludeDescendants != nil {
		include = *req.IncludeDescendants
	}
	ws, err := h.workspaces.LinkDepartment(r.Context(), actor, workspaceID, req.BitrixDepartmentID, include)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": ws})
}

func (h *WorkspaceHandler) UnlinkDepartment(w http.ResponseWriter, r *http.Request) {
	actor, _ := authn.UserID(r.Context())
	workspaceID := chi.URLParam(r, "workspaceID")
	deptID, err := strconv.ParseInt(chi.URLParam(r, "deptID"), 10, 64)
	if err != nil || deptID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	if err := h.workspaces.UnlinkDepartment(r.Context(), actor, workspaceID, deptID); err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *WorkspaceHandler) Apply(w http.ResponseWriter, r *http.Request) {
	actor, _ := authn.UserID(r.Context())
	res, err := h.workspaces.ApplyMemberships(r.Context(), actor)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": res})
}
