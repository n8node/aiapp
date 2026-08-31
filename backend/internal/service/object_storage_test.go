package service

import (
	"net/http"
	"testing"
)

func TestBrowserUploadHeadersDropsUnsafe(t *testing.T) {
	h := browserUploadHeaders("application/pdf", http.Header{
		"Host":           []string{"s3.regru.cloud"},
		"Content-Length": []string{"12"},
		"Content-Type":   []string{"text/plain"},
		"X-Amz-Date":     []string{"20260101T000000Z"},
		"Authorization":  []string{"AWS4-HMAC-SHA256 Credential=AKIA"},
	})
	if h["Content-Type"] != "application/pdf" {
		t.Fatalf("content-type %q", h["Content-Type"])
	}
	if _, ok := h["Host"]; ok {
		t.Fatal("host must not be sent by the browser")
	}
	if _, ok := h["Content-Length"]; ok {
		t.Fatal("content-length must not be sent")
	}
	if _, ok := h["Authorization"]; ok {
		t.Fatal("authorization belongs in the query string")
	}
	if h["X-Amz-Date"] != "20260101T000000Z" {
		t.Fatalf("x-amz-date %q", h["X-Amz-Date"])
	}
}
