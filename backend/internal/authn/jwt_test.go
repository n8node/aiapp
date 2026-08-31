package authn

import (
	"net/http"
	"testing"
)

func TestExtractTokenPrefersSessionCookieOverForeignBearer(t *testing.T) {
	r := &http.Request{Header: make(http.Header)}
	r.AddCookie(&http.Cookie{Name: AccessCookie, Value: "rigintel-session"})
	r.Header.Set("Authorization", "Bearer unsloth-jwt")
	if got := ExtractToken(r); got != "rigintel-session" {
		t.Fatalf("ExtractToken() = %q, want session cookie", got)
	}
}

func TestExtractTokenBearerWhenCookieMissing(t *testing.T) {
	r := &http.Request{Header: make(http.Header)}
	r.Header.Set("Authorization", "Bearer service-jwt")
	if got := ExtractToken(r); got != "service-jwt" {
		t.Fatalf("ExtractToken() = %q, want bearer", got)
	}
}
