package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/n8node/aiapp/internal/model"
)

func TestMaskSecret(t *testing.T) {
	if maskSecret("") != "" || maskSecret("ab") != "••••" {
		t.Fatal("short secrets")
	}
	got := maskSecret("abcdef")
	if strings.Contains(got, "cd") || !strings.HasPrefix(got, "ab") {
		t.Fatalf("got %q", got)
	}
}

func TestParseStorageEndpoint(t *testing.T) {
	got, err := parseStorageEndpoint("s3.ru1.storage.beget.cloud", true)
	if err != nil || got != "https://s3.ru1.storage.beget.cloud" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := parseStorageEndpoint("javascript:alert(1)", true); err == nil {
		t.Fatal("expected reject")
	}
	if _, err := parseStorageEndpoint("http://user:pass@minio:9000", false); err == nil {
		t.Fatal("expected reject userinfo")
	}
	if _, err := parseStorageEndpoint("http://169.254.169.254/", false); err == nil {
		t.Fatal("expected reject link-local")
	}
	if _, err := parseStorageEndpoint("http://minio:9000", false); err != nil {
		t.Fatal(err)
	}
}

func TestBuildCORSXMLContainsOrigins(t *testing.T) {
	xml := buildCORSXML([]string{"https://rigintel.ai"})
	if !strings.Contains(xml, "https://rigintel.ai") || !strings.Contains(xml, "PUT") {
		t.Fatalf("xml=%s", xml)
	}
}

func TestValidateStorageSettings(t *testing.T) {
	if err := validateStorageSettings(model.StorageAdminUpdateRequest{}); err != nil {
		t.Fatal(err)
	}
	err := validateStorageSettings(model.StorageAdminUpdateRequest{Enabled: true})
	if err == nil || !errors.Is(err, ErrInvalidStorageSettings) {
		t.Fatalf("got %v", err)
	}
}

func TestSanitizeS3ErrorHidesCredentials(t *testing.T) {
	got := sanitizeS3Error(errString("SecretAccessKey is wrong"))
	if strings.Contains(strings.ToLower(got), "secretaccesskey") {
		t.Fatalf("leaked: %s", got)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
