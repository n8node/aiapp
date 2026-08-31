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

func TestChunkStrings(t *testing.T) {
	got := chunkStrings([]string{"a", "b", "c", "d", "e"}, 2)
	if len(got) != 3 || len(got[2]) != 1 || got[2][0] != "e" {
		t.Fatalf("got %#v", got)
	}
	if chunkStrings(nil, 10) != nil {
		t.Fatal("empty")
	}
}

func TestUniqueS3Keys(t *testing.T) {
	got := uniqueS3Keys([]string{"", "a", "b", "a", "  "})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got %#v", got)
	}
}
