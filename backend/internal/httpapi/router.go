package httpapi

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/n8node/aiapp/internal/config"
)

const version = config.Version
const maxBodyBytes = 1 << 20

func NewRouter(cfg *config.Config, ping Pinger) http.Handler {
	health := NewHealthHandler(ping)
	status := NewStatusHandler(cfg.PublicAppURL)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(maxBytes(maxBodyBytes))
	r.Use(cors(cfg.CORSOrigins))
	r.Use(securityHeaders)

	r.Get("/live", health.Live)
	r.Get("/ready", health.Ready)
	r.Get("/health", health.Ready)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/status", status.ServeHTTP)
	})

	return r
}

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
