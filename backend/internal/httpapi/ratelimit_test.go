package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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
	h := limitAuth(login, reg, newRateLimiter(time.Minute, 40))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
