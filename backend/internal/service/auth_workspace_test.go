package service

import (
	"testing"

	"github.com/n8node/aiapp/internal/model"
)

func TestPickWorkspacePrefersRequested(t *testing.T) {
	list := []model.Workspace{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}
	got := pickWorkspace(list, "b")
	if got == nil || got.ID != "b" {
		t.Fatalf("got %#v", got)
	}
}

func TestPickWorkspaceFallsBackToFirst(t *testing.T) {
	list := []model.Workspace{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}
	got := pickWorkspace(list, "missing")
	if got == nil || got.ID != "a" {
		t.Fatalf("got %#v", got)
	}
}

func TestPickWorkspaceEmpty(t *testing.T) {
	if pickWorkspace(nil, "x") != nil {
		t.Fatal("expected nil")
	}
}
