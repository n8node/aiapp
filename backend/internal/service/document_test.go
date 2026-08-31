package service

import (
	"strings"
	"testing"

	"github.com/n8node/aiapp/internal/model"
)

func TestVersionReusableAndRetryable(t *testing.T) {
	if !versionReusable(model.DocStatusQueued) || !versionReusable(model.DocStatusPublished) {
		t.Fatal("reusable")
	}
	if versionReusable(model.DocStatusFailed) || versionReusable(model.DocStatusRejected) {
		t.Fatal("failed not reusable")
	}
	if !versionRetryable(model.DocStatusFailed) || versionRetryable(model.DocStatusPublished) {
		t.Fatal("retryable")
	}
}

func TestApplyReview(t *testing.T) {
	got, err := applyReview(model.DocStatusAwaitingReview, "approve")
	if err != nil || got != model.DocStatusPublished {
		t.Fatalf("approve %q %v", got, err)
	}
	got, err = applyReview(model.DocStatusAwaitingReview, "reject")
	if err != nil || got != model.DocStatusRejected {
		t.Fatalf("reject %q %v", got, err)
	}
	if _, err := applyReview(model.DocStatusQueued, "approve"); err != ErrDocumentInvalidState {
		t.Fatal("queued")
	}
	if _, err := applyReview(model.DocStatusAwaitingReview, "publish"); err != ErrInvalidInput {
		t.Fatal("bad action")
	}
}

func TestTruncateAndClipTitle(t *testing.T) {
	if truncateRunes("абв", 2) != "аб" {
		t.Fatal("runes")
	}
	if clipTitle("  ") != "Документ" {
		t.Fatal("empty title")
	}
	long := strings.Repeat("я", 300)
	if len([]rune(clipTitle(long))) != 255 {
		t.Fatal("clip")
	}
}

func TestPublicIngestError(t *testing.T) {
	if publicIngestError(ErrExtractUnavailable) != "extract_unavailable" {
		t.Fatal("unavailable")
	}
	if publicIngestError(ErrObjectTooLarge) != "object_too_large" {
		t.Fatal("too large")
	}
}
