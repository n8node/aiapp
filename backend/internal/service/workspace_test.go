package service

import (
	"testing"

	"github.com/n8node/aiapp/internal/model"
)

func ptr[T any](v T) *T { return &v }

func TestDepartmentCoverageClosestAncestorWins(t *testing.T) {
	depts := []model.BitrixDepartment{
		{BitrixID: 1, Name: "Root"},
		{BitrixID: 36, ParentBitrixID: ptr(int64(1)), Name: "R&D"},
		{BitrixID: 38, ParentBitrixID: ptr(int64(36)), Name: "AI"},
		{BitrixID: 16, ParentBitrixID: ptr(int64(1)), Name: "Finance"},
	}
	links := []model.WorkspaceBitrixLink{
		{WorkspaceID: "ws-root", BitrixDepartmentID: 1, IncludeDescendants: true},
		{WorkspaceID: "ws-rd", BitrixDepartmentID: 36, IncludeDescendants: true},
	}
	got := DepartmentCoverage(depts, links)
	if got[1] != "ws-root" || got[36] != "ws-rd" || got[38] != "ws-rd" || got[16] != "ws-root" {
		t.Fatalf("coverage=%v", got)
	}
}

func TestDepartmentCoverageExplicitChildBlocksParent(t *testing.T) {
	depts := []model.BitrixDepartment{
		{BitrixID: 36, Name: "R&D"},
		{BitrixID: 38, ParentBitrixID: ptr(int64(36)), Name: "AI"},
	}
	links := []model.WorkspaceBitrixLink{
		{WorkspaceID: "ws-rd", BitrixDepartmentID: 36, IncludeDescendants: true},
		{WorkspaceID: "ws-ai", BitrixDepartmentID: 38, IncludeDescendants: false},
	}
	got := DepartmentCoverage(depts, links)
	if got[38] != "ws-ai" {
		t.Fatalf("child should stay explicit, got %v", got)
	}
}

func TestDesiredMembershipsHeadIsManager(t *testing.T) {
	deps := []model.BitrixDepartment{
		{BitrixID: 36, HeadBitrixID: ptr(int64(6))},
	}
	users := []model.BitrixUser{
		{BitrixID: 6, Email: "head@rigintel.ai", Active: true, DepartmentIDs: []int64{36}},
		{BitrixID: 7, Email: "emp@rigintel.ai", Active: true, DepartmentIDs: []int64{36}},
	}
	local := map[string]string{
		"head@rigintel.ai": "u-head",
		"emp@rigintel.ai":  "u-emp",
	}
	coverage := map[int64]string{36: "ws-rd"}
	got := desiredBitrixMemberships(users, local, coverage, deps)
	if got["ws-rd"]["u-head"] != roleDepartmentManager {
		t.Fatalf("head role=%q", got["ws-rd"]["u-head"])
	}
	if got["ws-rd"]["u-emp"] != roleEmployee {
		t.Fatalf("emp role=%q", got["ws-rd"]["u-emp"])
	}
}
