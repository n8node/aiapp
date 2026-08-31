package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/n8node/aiapp/internal/authn"
	"github.com/n8node/aiapp/internal/password"
	"github.com/n8node/aiapp/internal/service"
)

type AuthHandler struct {
	auth   *service.AuthService
	tokens *authn.JWT
}

func NewAuthHandler(auth *service.AuthService, tokens *authn.JWT) *AuthHandler {
	return &AuthHandler{auth: auth, tokens: tokens}
}

type registerRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	Name       string `json:"name"`
	InviteCode string `json:"invite_code"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	TotpCode string `json:"totp_code"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	if err := password.Validate(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, "password_policy", "Проверьте пароль (мин. 8 символов, заглавные, цифры, спецсимвол)")
		return
	}
	res, err := h.auth.Register(r.Context(), req.Email, req.Password, req.Name, req.InviteCode)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.tokens.SetCookie(w, res.Token)
	writeJSON(w, http.StatusCreated, authPayload(res))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	res, err := h.auth.Login(r.Context(), req.Email, req.Password, req.TotpCode)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	h.tokens.SetCookie(w, res.Token)
	writeJSON(w, http.StatusOK, authPayload(res))
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if raw := authn.ExtractToken(r); raw != "" {
		if _, sessionID, err := h.tokens.Parse(raw); err == nil {
			h.auth.Logout(r.Context(), sessionID)
		}
	}
	h.tokens.ClearCookie(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := authn.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
		return
	}
	sessionID, _ := authn.SessionID(r.Context())
	user, workspaces, active, err := h.auth.Me(r.Context(), userID, sessionID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"user":             user,
			"workspace":        active,
			"active_workspace": active,
			"workspaces":       workspaces,
		},
	})
}

func (h *AuthHandler) SwitchWorkspace(w http.ResponseWriter, r *http.Request) {
	userID, _ := authn.UserID(r.Context())
	sessionID, _ := authn.SessionID(r.Context())
	var req struct {
		WorkspaceID string `json:"workspace_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	ws, err := h.auth.SwitchWorkspace(r.Context(), userID, sessionID, req.WorkspaceID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"workspace": ws}})
}

func (h *AuthHandler) VerifyInvite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InviteCode string `json:"invite_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	if err := h.auth.VerifyInvite(r.Context(), req.InviteCode); err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]bool{"valid": true}})
}

func (h *AuthHandler) SetupTotp(w http.ResponseWriter, r *http.Request) {
	userID, _ := authn.UserID(r.Context())
	secret, url, err := h.auth.SetupTotp(r.Context(), userID)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]string{"secret": secret, "otpauth_url": url},
	})
}

func (h *AuthHandler) ConfirmTotp(w http.ResponseWriter, r *http.Request) {
	userID, _ := authn.UserID(r.Context())
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	if err := h.auth.ConfirmTotp(r.Context(), userID, req.Code); err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]bool{"totp_enabled": true}})
}

func (h *AuthHandler) AdminUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	var blocked *bool
	if v := r.URL.Query().Get("is_blocked"); v == "true" || v == "false" {
		b := v == "true"
		blocked = &b
	}
	var admin *bool
	if v := r.URL.Query().Get("is_platform_admin"); v == "true" || v == "false" {
		b := v == "true"
		admin = &b
	}
	users, total, err := h.auth.ListUsers(r.Context(), q, blocked, admin, 50, 0)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"users": users, "total": total}})
}

func (h *AuthHandler) AdminBlockUser(w http.ResponseWriter, r *http.Request) {
	actor, _ := authn.UserID(r.Context())
	id := chi.URLParam(r, "userID")
	var req struct {
		Blocked bool `json:"is_blocked"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	if err := h.auth.SetUserBlocked(r.Context(), actor, id, req.Blocked); err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *AuthHandler) AdminIssueInvites(w http.ResponseWriter, r *http.Request) {
	actor, _ := authn.UserID(r.Context())
	var req struct {
		Count int `json:"count"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Count == 0 {
		req.Count = 1
	}
	issued, err := h.auth.IssueInvites(r.Context(), actor, req.Count)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": map[string]any{"invites": issued}})
}

func (h *AuthHandler) AdminListInvites(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f, err := parseInviteListFilter(q)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте параметры фильтра")
		return
	}
	invites, total, err := h.auth.ListInvites(r.Context(), f)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"invites": invites, "total": total}})
}

func (h *AuthHandler) AdminDeleteInvites(w http.ResponseWriter, r *http.Request) {
	actor, _ := authn.UserID(r.Context())
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	n, err := h.auth.DeleteInvites(r.Context(), actor, req.IDs)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"deleted": n}})
}

func (h *AuthHandler) AdminRevokeInvite(w http.ResponseWriter, r *http.Request) {
	actor, _ := authn.UserID(r.Context())
	id := chi.URLParam(r, "inviteID")
	if err := h.auth.RevokeInvite(r.Context(), actor, id); err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *AuthHandler) AdminGetDomains(w http.ResponseWriter, r *http.Request) {
	domains, err := h.auth.AllowedDomains(r.Context())
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"domains": domains}})
}

func (h *AuthHandler) AdminSetDomains(w http.ResponseWriter, r *http.Request) {
	actor, _ := authn.UserID(r.Context())
	var req struct {
		Domains []string `json:"domains"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Проверьте введённые данные")
		return
	}
	domains, err := h.auth.SetAllowedDomains(r.Context(), actor, req.Domains)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"domains": domains}})
}

func authPayload(res *service.AuthResult) map[string]any {
	return map[string]any{
		"data": map[string]any{
			"user":             res.User,
			"workspace":        res.Workspace,
			"active_workspace": res.Workspace,
			"workspaces":       res.Workspaces,
		},
	}
}
