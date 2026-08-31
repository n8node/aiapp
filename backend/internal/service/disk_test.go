package service

import (
	"strings"
	"testing"
)

func TestValidateDiskName(t *testing.T) {
	if err := validateDiskName("Договор"); err != nil {
		t.Fatal(err)
	}
	if err := validateDiskName("a/b"); err == nil {
		t.Fatal("slash")
	}
	if err := validateDiskName(""); err == nil {
		t.Fatal("empty")
	}
}

func TestDuplicateName(t *testing.T) {
	if got := duplicateName("act.pdf"); got != "act (копия).pdf" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildDiskS3Key(t *testing.T) {
	got := buildDiskS3Key("11111111-1111-1111-1111-111111111111")
	if !strings.HasPrefix(got, "workspaces/11111111-1111-1111-1111-111111111111/files/") {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "..") || strings.Contains(got, " ") {
		t.Fatal("unsafe")
	}
}
