package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/n8node/aiapp/internal/authn"
	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/service"
)

type DiskHandler struct {
	disk *service.DiskService
}

func NewDiskHandler(disk *service.DiskService) *DiskHandler {
	return &DiskHandler{disk: disk}
}

func (h *DiskHandler) actor(r *http.Request) (userID, sessionID string, ok bool) {
	userID, ok = authn.UserID(r.Context())
	if !ok {
		return "", "", false
	}
	sessionID, ok = authn.SessionID(r.Context())
	return userID, sessionID, ok
}

func (h *DiskHandler) UploadInit(w http.ResponseWriter, r *http.Request) {
	if h.disk == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	userID, sessionID, ok := h.actor(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var req model.FileUploadInitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	out, err := h.disk.UploadInit(r.Context(), userID, sessionID, req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *DiskHandler) UploadComplete(w http.ResponseWriter, r *http.Request) {
	if h.disk == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	userID, sessionID, ok := h.actor(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var req model.FileUploadCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	file, err := h.disk.UploadComplete(r.Context(), userID, sessionID, req.UploadSessionToken)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": file})
}

func (h *DiskHandler) ListFiles(w http.ResponseWriter, r *http.Request) {
	if h.disk == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Не удалось выполнить запрос")
		return
	}
	userID, sessionID, ok := h.actor(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var folderID *string
	if v := r.URL.Query().Get("folder_id"); v != "" {
		folderID = &v
	}
	section := r.URL.Query().Get("section")
	if section == "" {
		section = "my-files"
	}
	items, err := h.disk.ListFiles(r.Context(), userID, sessionID, section, folderID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"files": items}})
}

func (h *DiskHandler) GetFile(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.disk == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	file, err := h.disk.GetFile(r.Context(), userID, sessionID, chi.URLParam(r, "fileID"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": file})
}

func (h *DiskHandler) PatchFile(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.disk == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var req model.DiskFilePatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	file, err := h.disk.PatchFile(r.Context(), userID, sessionID, chi.URLParam(r, "fileID"), req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": file})
}

func (h *DiskHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.disk == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	if err := h.disk.DeleteFile(r.Context(), userID, sessionID, chi.URLParam(r, "fileID")); err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *DiskHandler) CopyFile(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.disk == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var req struct {
		FolderID *string `json:"folder_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	file, err := h.disk.CopyFile(r.Context(), userID, sessionID, chi.URLParam(r, "fileID"), req.FolderID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": file})
}

func (h *DiskHandler) Download(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.disk == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	inline := r.URL.Query().Get("disposition") == "inline"
	url, err := h.disk.DownloadURL(r.Context(), userID, sessionID, chi.URLParam(r, "fileID"), inline)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": model.DiskDownloadResponse{URL: url}})
}

func (h *DiskHandler) BulkFiles(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.disk == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var req model.DiskBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	res, err := h.disk.BulkFiles(r.Context(), userID, sessionID, req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": res})
}

func (h *DiskHandler) ListFolders(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.disk == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	q := r.URL.Query()
	var parentID *string
	if v := q.Get("parent_id"); v != "" {
		parentID = &v
	}
	items, err := h.disk.ListFolders(r.Context(), userID, sessionID, parentID, q.Get("scope") == "all")
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"folders": items}})
}

func (h *DiskHandler) CreateFolder(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.disk == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var req model.DiskFolderWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	folder, err := h.disk.CreateFolder(r.Context(), userID, sessionID, req.Name, req.ParentID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": folder})
}

func (h *DiskHandler) Breadcrumbs(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.disk == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	crumbs, err := h.disk.Breadcrumbs(r.Context(), userID, sessionID, chi.URLParam(r, "folderID"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"breadcrumbs": crumbs}})
}

func (h *DiskHandler) PatchFolder(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.disk == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var req model.DiskFolderPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	folder, err := h.disk.PatchFolder(r.Context(), userID, sessionID, chi.URLParam(r, "folderID"), req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": folder})
}

func (h *DiskHandler) DeleteFolder(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.disk == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	if err := h.disk.DeleteFolder(r.Context(), userID, sessionID, chi.URLParam(r, "folderID")); err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *DiskHandler) BulkFolders(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.disk == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var req model.DiskBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	res, err := h.disk.BulkFolders(r.Context(), userID, sessionID, req)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": res})
}

func (h *DiskHandler) ListTrash(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.disk == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	files, folders, err := h.disk.ListTrash(r.Context(), userID, sessionID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"files": files, "folders": folders}})
}

func (h *DiskHandler) RestoreTrash(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.disk == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	var req model.DiskTrashRestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	if err := h.disk.RestoreTrash(r.Context(), userID, sessionID, req.FileIDs, req.FolderIDs); err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *DiskHandler) EmptyTrash(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.disk == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	if err := h.disk.EmptyTrash(r.Context(), userID, sessionID); err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]bool{"ok": true}})
}

func (h *DiskHandler) PermanentDelete(w http.ResponseWriter, r *http.Request) {
	userID, sessionID, ok := h.actor(r)
	if !ok || h.disk == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = "file"
	}
	if err := h.disk.PermanentDelete(r.Context(), userID, sessionID, chi.URLParam(r, "id"), kind); err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]bool{"ok": true}})
}
