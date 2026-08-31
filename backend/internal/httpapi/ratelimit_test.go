package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/n8node/aiapp/internal/authn"
)

func TestRateLimiterAllowsThenBlocks(t *testing.T) {
	l := newRateLimiter(time.Minute, 2)
	if !l.allow("a") || !l.allow("a") {
		t.Fatal("first two should pass")
	}
	if l.allow("a") {
		t.Fatal("third should block")
	}
	if !l.allow("b") {
		t.Fatal("other key should pass")
	}
}

func TestLimitAuthLogin(t *testing.T) {
	login := newRateLimiter(time.Minute, 1)
	reg := newRateLimiter(time.Minute, 5)
	h := limitAuth(login, reg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("first login got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second login got %d", rec.Code)
	}
}

func TestLimitDiskUploadPerUser(t *testing.T) {
	upload := newRateLimiter(time.Minute, 1)
	h := limitDiskUpload(upload)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/disk/files/upload/init", nil)
	req = req.WithContext(authn.WithUser(req.Context(), "user-a", "sess"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("first got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second got %d", rec.Code)
	}
	reqB := httptest.NewRequest(http.MethodPost, "/api/v1/disk/files/upload/init", nil)
	reqB = reqB.WithContext(authn.WithUser(reqB.Context(), "user-b", "sess"))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, reqB)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("other user got %d", rec.Code)
	}
}
