package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/n8node/aiapp/internal/config"
)

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		PublicAppURL: "https://rigintel.ai/app",
		CORSOrigins:  []string{"https://rigintel.ai"},
	}
	handler := NewRouter(Dependencies{Config: cfg, Ping: stubPinger{}})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("unexpected CORS allow for unknown origin")
	}
}

func TestCORSAllowsExactOrigin(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		PublicAppURL: "https://rigintel.ai/app",
		CORSOrigins:  []string{"https://rigintel.ai"},
	}
	handler := NewRouter(Dependencies{Config: cfg, Ping: stubPinger{}})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	req.Header.Set("Origin", "https://rigintel.ai")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://rigintel.ai" {
		t.Fatalf("missing exact origin allow")
	}
}

func TestMatchOrigin(t *testing.T) {
	t.Parallel()
	if MatchOrigin([]string{"https://rigintel.ai"}, "*") {
		t.Fatal("wildcard must not match")
	}
	if !MatchOrigin([]string{"https://rigintel.ai"}, "https://rigintel.ai") {
		t.Fatal("exact origin should match")
	}
}
