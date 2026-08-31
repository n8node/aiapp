package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubPinger struct {
	err error
}

func (s stubPinger) Ping(context.Context) error {
	return s.err
}

func TestHealthReady(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		pingErr    error
		wantCode   int
		wantStatus string
		wantPG     string
	}{
		{name: "ok", wantCode: http.StatusOK, wantStatus: "ok", wantPG: "ok"},
		{name: "db down", pingErr: errors.New("unavailable"), wantCode: http.StatusServiceUnavailable, wantStatus: "degraded", wantPG: "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := NewHealthHandler(stubPinger{err: tt.pingErr})
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()
			h.Ready(rec, req)

			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantCode)
			}

			var body healthResponse
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Status != tt.wantStatus {
				t.Fatalf("body.status = %q, want %q", body.Status, tt.wantStatus)
			}
			if body.Postgres != tt.wantPG {
				t.Fatalf("body.postgres = %q, want %q", body.Postgres, tt.wantPG)
			}
			if body.App != "rigintel" {
				t.Fatalf("body.app = %q", body.App)
			}
		})
	}
}

func TestHealthLiveIgnoresDatabase(t *testing.T) {
	t.Parallel()
	h := NewHealthHandler(stubPinger{err: errors.New("unavailable")})
	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	rec := httptest.NewRecorder()
	h.Live(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
