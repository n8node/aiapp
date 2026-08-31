package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/n8node/aiapp/internal/authn"
	"github.com/n8node/aiapp/internal/config"
	"github.com/n8node/aiapp/internal/service"
)

type Dependencies struct {
	Config     *config.Config
	Ping       Pinger
	Auth       *service.AuthService
	Bitrix     *service.BitrixService
	Workspaces *service.WorkspaceService
	Storage    *service.StorageSettingsService
	Disk       *service.DiskService
	Tokens     *authn.JWT
}

func NewRouter(deps Dependencies) http.Handler {
	health := NewHealthHandler(deps.Ping)
	status := NewStatusHandler(deps.Config.PublicAppURL)
	authH := NewAuthHandler(deps.Auth, deps.Tokens)
	bitrixH := NewBitrixHandler(deps.Bitrix, deps.Workspaces)
	workspaceH := NewWorkspaceHandler(deps.Workspaces)
	storageH := NewStorageHandler(deps.Storage)
	diskH := NewDiskHandler(deps.Disk)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(maxBytes(maxBodyBytes))
	r.Use(cors(deps.Config.CORSOrigins))
	r.Use(csrfOrigin(deps.Config.CORSOrigins))
	r.Use(securityHeaders)
	r.Use(limitAuth(newRateLimiter(15*time.Minute, 20), newRateLimiter(15*time.Minute, 10)))

	r.Get("/live", health.Live)
	r.Get("/ready", health.Ready)
	r.Get("/health", health.Ready)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/status", status.ServeHTTP)
		r.Post("/auth/register", authH.Register)
		r.Post("/auth/login", authH.Login)
		r.Post("/auth/logout", authH.Logout)
		r.Post("/auth/invite/verify", authH.VerifyInvite)

		r.Group(func(r chi.Router) {
			r.Use(requireAuth(deps.Tokens, deps.Auth))
			r.Get("/auth/me", authH.Me)
			r.Post("/auth/2fa/setup", authH.SetupTotp)
			r.Post("/auth/2fa/confirm", authH.ConfirmTotp)

			r.Group(func(r chi.Router) {
				r.Use(requireTotp(deps.Auth))
				r.Use(limitDiskUpload(newRateLimiter(15*time.Minute, 500)))
				r.Put("/auth/workspace", authH.SwitchWorkspace)
				r.Get("/disk/files", diskH.ListFiles)
				r.Post("/disk/files/upload/init", diskH.UploadInit)
				r.Post("/disk/files/upload/complete", diskH.UploadComplete)
				r.Post("/disk/files/bulk", diskH.BulkFiles)
				r.Get("/disk/files/{fileID}", diskH.GetFile)
				r.Patch("/disk/files/{fileID}", diskH.PatchFile)
				r.Delete("/disk/files/{fileID}", diskH.DeleteFile)
				r.Post("/disk/files/{fileID}/copy", diskH.CopyFile)
				r.Get("/disk/files/{fileID}/download", diskH.Download)
				r.Get("/disk/folders", diskH.ListFolders)
				r.Post("/disk/folders", diskH.CreateFolder)
				r.Post("/disk/folders/bulk", diskH.BulkFolders)
				r.Get("/disk/folders/{folderID}/breadcrumbs", diskH.Breadcrumbs)
				r.Patch("/disk/folders/{folderID}", diskH.PatchFolder)
				r.Delete("/disk/folders/{folderID}", diskH.DeleteFolder)
				r.Get("/disk/trash", diskH.ListTrash)
				r.Post("/disk/trash/restore", diskH.RestoreTrash)
				r.Post("/disk/trash/empty", diskH.EmptyTrash)
				r.Delete("/disk/trash/{id}", diskH.PermanentDelete)
				r.Group(func(r chi.Router) {
					r.Use(requireAdmin(deps.Auth))
					r.Get("/admin/users", authH.AdminUsers)
					r.Post("/admin/users/{userID}/block", authH.AdminBlockUser)
					r.Get("/admin/invites", authH.AdminListInvites)
					r.Post("/admin/invites", authH.AdminIssueInvites)
					r.Post("/admin/invites/delete", authH.AdminDeleteInvites)
					r.Post("/admin/invites/{inviteID}/revoke", authH.AdminRevokeInvite)
					r.Get("/admin/auth-domains", authH.AdminGetDomains)
					r.Put("/admin/auth-domains", authH.AdminSetDomains)
					r.Get("/admin/bitrix", bitrixH.Status)
					r.Put("/admin/bitrix", bitrixH.Save)
					r.Post("/admin/bitrix/test", bitrixH.Test)
					r.Post("/admin/bitrix/sync", bitrixH.Sync)
					r.Get("/admin/bitrix/departments", bitrixH.Departments)
					r.Get("/admin/bitrix/users", bitrixH.Users)
					r.Get("/admin/workspaces", workspaceH.List)
					r.Post("/admin/workspaces", workspaceH.CreateFromDepartment)
					r.Post("/admin/workspaces/apply", workspaceH.Apply)
					r.Post("/admin/workspaces/{workspaceID}/bitrix", workspaceH.LinkDepartment)
					r.Delete("/admin/workspaces/{workspaceID}/bitrix/{deptID}", workspaceH.UnlinkDepartment)
					r.Get("/admin/storage-settings", storageH.Get)
					r.Put("/admin/storage-settings", storageH.Save)
					r.Post("/admin/storage-settings/test", storageH.Test)
				})
			})
		})
	})

	return r
}

func requireAuth(tokens *authn.JWT, auth *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := authn.ExtractToken(r)
			if raw == "" {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
				return
			}
			userID, sessionID, err := tokens.Parse(raw)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
				return
			}
			if _, err := auth.RequireSession(r.Context(), userID, sessionID); err != nil {
				writeAuthError(w, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(authn.WithUser(r.Context(), userID, sessionID)))
		})
	}
}

func requireTotp(auth *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := authn.UserID(r.Context())
			if !ok {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Не авторизован")
				return
			}
			user, err := auth.RequireUser(r.Context(), userID)
			if err != nil {
				writeAuthError(w, err)
				return
			}
			if !user.TotpEnabled {
				writeAuthError(w, service.ErrTotpSetupRequired)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func requireAdmin(auth *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, _ := authn.UserID(r.Context())
			user, err := auth.RequireUser(r.Context(), userID)
			if err != nil {
				writeAuthError(w, err)
				return
			}
			if !user.IsPlatformAdmin {
				writeError(w, http.StatusForbidden, "forbidden", "Недостаточно прав")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func csrfOrigin(allowlist []string) func(http.Handler) http.Handler {
	allowed := map[string]struct{}{}
	for _, origin := range allowlist {
		if origin != "" {
			allowed[origin] = struct{}{}
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}
			if _, ok := allowed[origin]; !ok {
				writeError(w, http.StatusForbidden, "csrf", "Запрос отклонён")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

const version = config.Version
const maxBodyBytes = 1 << 20

func maxBytes(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func cors(allowlist []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowlist))
	for _, origin := range allowlist {
		if origin != "" {
			allowed[origin] = struct{}{}
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if _, ok := allowed[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				}
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func requestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-Id"); id != "" {
		return id
	}
	if id := middleware.GetReqID(r.Context()); id != "" {
		return id
	}
	return ""
}

func MatchOrigin(allowlist []string, origin string) bool {
	for _, item := range allowlist {
		if strings.EqualFold(item, origin) {
			return true
		}
	}
	return false
}
