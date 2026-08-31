package bitrix

import (
	"strings"
	"testing"
)

func TestErrorCodeIgnoresEmpty(t *testing.T) {
	for _, raw := range []string{"", "null", `""`, "false", "0"} {
		if got := errorCode([]byte(raw)); got != "" {
			t.Fatalf("errorCode(%s)=%q", raw, got)
		}
	}
}

func TestErrorCodeReadsString(t *testing.T) {
	if got := errorCode([]byte(`"insufficient_scope"`)); got != "insufficient_scope" {
		t.Fatalf("got %q", got)
	}
}

func TestPublicBitrixErrorScope(t *testing.T) {
	got := publicBitrixError("ERROR_METHOD_NOT_FOUND", "Method not found!")
	if !strings.Contains(got, "department") && !strings.Contains(got, "Структура") {
		t.Fatalf("expected department hint, got %q", got)
	}
}

func TestParseEnvelopeRejectsHTML(t *testing.T) {
	_, err := parseEnvelope([]byte("<html>nope</html>"))
	if err == nil {
		t.Fatal("expected error")
	}
	if PublicError(err) == "" {
		t.Fatal("public message required")
	}
}
