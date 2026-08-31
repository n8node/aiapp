package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/n8node/aiapp/internal/authn"
)

type rateLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	window time.Duration
	max    int
}

func newRateLimiter(window time.Duration, max int) *rateLimiter {
	return &rateLimiter{hits: map[string][]time.Time{}, window: window, max: max}
}

func (l *rateLimiter) allow(key string) bool {
	now := time.Now()
	cut := now.Add(-l.window)
	l.mu.Lock()
	defer l.mu.Unlock()
	xs := l.hits[key]
	n := 0
	for _, t := range xs {
		if t.After(cut) {
			xs[n] = t
			n++
		}
	}
	xs = xs[:n]
	if len(xs) >= l.max {
		l.hits[key] = xs
		return false
	}
	l.hits[key] = append(xs, now)
	return true
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func limitAuth(login, register *rateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}
			ip := clientIP(r)
			switch r.URL.Path {
			case "/api/v1/auth/login":
				if !login.allow(ip) {
					writeError(w, http.StatusTooManyRequests, "rate_limited", "Слишком много попыток, подождите")
					return
				}
			case "/api/v1/auth/register", "/api/v1/auth/invite/verify":
				if !register.allow(ip) {
					writeError(w, http.StatusTooManyRequests, "rate_limited", "Слишком много попыток, подождите")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func limitDiskUpload(upload *rateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}
			path := r.URL.Path
			kind := ""
			switch {
			case path == "/api/v1/disk/files/upload/init":
				kind = "upload"
			case strings.HasSuffix(path, "/extract") && strings.Contains(path, "/disk/files/"):
				kind = "extract"
			case path == "/api/v1/documents/from-file":
				kind = "ingest"
			case strings.HasSuffix(path, "/retry") && strings.Contains(path, "/documents/"):
				kind = "ingest"
			default:
				next.ServeHTTP(w, r)
				return
			}
			key, ok := authn.UserID(r.Context())
			if !ok || key == "" {
				key = clientIP(r)
			}
			if !upload.allow(key + ":" + kind) {
				w.Header().Set("Retry-After", "5")
				msg := "Слишком много загрузок, подождите"
				if kind == "extract" {
					msg = "Слишком много распаковок, подождите"
				}
				if kind == "ingest" {
					msg = "Слишком много запросов на обработку, подождите"
				}
				writeError(w, http.StatusTooManyRequests, "rate_limited", msg)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
