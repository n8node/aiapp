package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/repository"
)

const (
	roleEmployee          = "employee"
	roleDepartmentManager = "department_manager"
)

var (
	ErrWorkspaceNotFound = errors.New("workspace not found")
	ErrDepartmentUnknown = errors.New("bitrix department unknown")
	ErrDepartmentMapped  = errors.New("bitrix department already mapped")
)

var workspaceSlugSanitizer = regexp.MustCompile(`[^a-z0-9-]+`)

type WorkspaceService struct {
	workspaces *repository.WorkspaceRepository
	users      *repository.UserRepository
	bitrix     *repository.BitrixRepository
	audit      *repository.AuditRepository
}

func NewWorkspaceService(
	workspaces *repository.WorkspaceRepository,
	users *repository.UserRepository,
	bitrix *repository.BitrixRepository,
	audit *repository.AuditRepository,
) *WorkspaceService {
	return &WorkspaceService{workspaces: workspaces, users: users, bitrix: bitrix, audit: audit}
}

func (s *WorkspaceService) List(ctx context.Context) ([]model.WorkspaceAdmin, error) {
	return s.workspaces.ListAdmin(ctx)
}

func (s *WorkspaceService) EnrichDepartments(ctx context.Context, deps []model.BitrixDepartment) ([]model.BitrixDepartment, error) {
	links, err := s.workspaces.ListLinks(ctx)
	if err != nil {
		return nil, err
	}
	coverage := DepartmentCoverage(deps, links)
	byDept := map[int64]model.WorkspaceBitrixLink{}
	for _, l := range links {
		byDept[l.BitrixDepartmentID] = l
	}
	nameByID := map[int64]string{}
	parentOf := map[int64]*int64{}
	for _, d := range deps {
		nameByID[d.BitrixID] = d.Name
		parentOf[d.BitrixID] = d.ParentBitrixID
	}
	nameByWS := map[string]string{}
	for _, l := range links {
		nameByWS[l.WorkspaceID] = l.WorkspaceName
	}
	for i := range deps {
		if link, ok := byDept[deps[i].BitrixID]; ok {
			wsID := link.WorkspaceID
			deps[i].WorkspaceID = &wsID
			if link.WorkspaceName != "" {
				name := link.WorkspaceName
				deps[i].WorkspaceName = &name
			}
			inc := link.IncludeDescendants
			deps[i].IncludeDescendants = &inc
			continue
		}
		wsID, ok := coverage[deps[i].BitrixID]
		if !ok {
			continue
		}
		deps[i].WorkspaceID = &wsID
		deps[i].WorkspaceInherited = true
		if name := nameByWS[wsID]; name != "" {
			deps[i].WorkspaceName = &name
		}
		deps[i].InheritedFrom = ancestorLinkName(deps[i].BitrixID, parentOf, byDept, nameByID)
	}
	return deps, nil
}

func ancestorLinkName(id int64, parent map[int64]*int64, links map[int64]model.WorkspaceBitrixLink, names map[int64]string) string {
	seen := map[int64]struct{}{}
	cur := parent[id]
	for cur != nil {
		if _, loop := seen[*cur]; loop {
			break
		}
		seen[*cur] = struct{}{}
		if _, ok := links[*cur]; ok {
			if n := names[*cur]; n != "" {
				return n
			}
			return "родительский отдел"
		}
		cur = parent[*cur]
	}
	return ""
}

func (s *WorkspaceService) CreateFromDepartment(ctx context.Context, actorID string, deptID int64, includeDescendants bool, name string) (*model.WorkspaceAdmin, error) {
	deps, err := s.bitrix.ListDepartments(ctx)
	if err != nil {
		return nil, err
	}
	var dept *model.BitrixDepartment
	for i := range deps {
		if deps[i].BitrixID == deptID {
			dept = &deps[i]
			break
		}
	}
	if dept == nil {
		return nil, ErrDepartmentUnknown
	}
	links, err := s.workspaces.ListLinks(ctx)
	if err != nil {
		return nil, err
	}
	for _, l := range links {
		if l.BitrixDepartmentID == deptID {
			return nil, ErrDepartmentMapped
		}
	}
	if strings.TrimSpace(name) == "" {
		name = strings.TrimSpace(dept.Name)
	}
	if name == "" {
		name = "Отдел " + strconv.FormatInt(deptID, 10)
	}
	slug, err := s.uniqueSlug(ctx, slugFromDepartment(name, deptID))
	if err != nil {
		return nil, err
	}
	ws, err := s.workspaces.CreateWithOwner(ctx, name, slug, actorID)
	if err != nil {
		return nil, err
	}
	if err := s.workspaces.LinkDepartment(ctx, ws.ID, deptID, includeDescendants); err != nil {
		return nil, err
	}
	s.audit.Write(ctx, actorID, "workspace.bitrix.create", fmt.Sprintf("%s:%d", ws.ID, deptID))
	return s.getAdmin(ctx, ws.ID)
}

func (s *WorkspaceService) LinkDepartment(ctx context.Context, actorID, workspaceID string, deptID int64, includeDescendants bool) (*model.WorkspaceAdmin, error) {
	if _, err := s.workspaces.GetByID(ctx, workspaceID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrWorkspaceNotFound
		}
		return nil, err
	}
	deps, err := s.bitrix.ListDepartments(ctx)
	if err != nil {
		return nil, err
	}
	found := false
	for _, d := range deps {
		if d.BitrixID == deptID {
			found = true
			break
		}
	}
	if !found {
		return nil, ErrDepartmentUnknown
	}
	if err := s.workspaces.LinkDepartment(ctx, workspaceID, deptID, includeDescendants); err != nil {
		return nil, err
	}
	s.audit.Write(ctx, actorID, "workspace.bitrix.link", fmt.Sprintf("%s:%d", workspaceID, deptID))
	return s.getAdmin(ctx, workspaceID)
}

func (s *WorkspaceService) UnlinkDepartment(ctx context.Context, actorID, workspaceID string, deptID int64) error {
	if err := s.workspaces.UnlinkDepartment(ctx, workspaceID, deptID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrWorkspaceNotFound
		}
		return err
	}
	s.audit.Write(ctx, actorID, "workspace.bitrix.unlink", fmt.Sprintf("%s:%d", workspaceID, deptID))
	return nil
}

func (s *WorkspaceService) ApplyMemberships(ctx context.Context, actorID string) (*model.MembershipApplyResult, error) {
	deps, err := s.bitrix.ListDepartments(ctx)
	if err != nil {
		return nil, err
	}
	links, err := s.workspaces.ListLinks(ctx)
	if err != nil {
		return nil, err
	}
	users, err := s.bitrix.ListAllUsers(ctx)
	if err != nil {
		return nil, err
	}
	emails := make([]string, 0, len(users))
	for _, u := range users {
		emails = append(emails, u.Email)
	}
	localIDs, err := s.users.IDsByEmails(ctx, emails)
	if err != nil {
		return nil, err
	}
	coverage := DepartmentCoverage(deps, links)
	desired := desiredBitrixMemberships(users, localIDs, coverage, deps)
	current, err := s.workspaces.ListBitrixMembers(ctx)
	if err != nil {
		return nil, err
	}
	res := &model.MembershipApplyResult{Workspaces: uniqueWorkspaceCount(desired)}
	seenLocal := map[string]struct{}{}
	for _, u := range users {
		email := strings.ToLower(strings.TrimSpace(u.Email))
		if email == "" || !u.Active {
			continue
		}
		if _, ok := localIDs[email]; !ok {
			res.Unmatched++
			continue
		}
		if _, ok := seenLocal[email]; ok {
			continue
		}
		seenLocal[email] = struct{}{}
		res.Matched++
	}

	type key struct{ ws, user string }
	have := map[key]repository.WorkspaceMember{}
	for _, m := range current {
		have[key{m.WorkspaceID, m.UserID}] = m
	}
	want := map[key]string{}
	for wsID, members := range desired {
		for userID, role := range members {
			want[key{wsID, userID}] = role
		}
	}
	for k, role := range want {
		existing, ok := have[k]
		if !ok {
			local, err := s.workspaces.GetMember(ctx, k.ws, k.user)
			if err == nil && local.Source == "local" {
				continue
			}
			if err := s.workspaces.UpsertBitrixMember(ctx, k.ws, k.user, role); err != nil {
				return nil, err
			}
			res.Added++
			continue
		}
		if existing.Role != role {
			if err := s.workspaces.UpsertBitrixMember(ctx, k.ws, k.user, role); err != nil {
				return nil, err
			}
			res.Updated++
		}
	}
	for k := range have {
		if _, ok := want[k]; ok {
			continue
		}
		if err := s.workspaces.DeleteBitrixMember(ctx, k.ws, k.user); err != nil {
			return nil, err
		}
		res.Removed++
	}
	s.audit.Write(ctx, actorID, "workspace.bitrix.apply", fmt.Sprintf("added=%d updated=%d removed=%d", res.Added, res.Updated, res.Removed))
	return res, nil
}

func (s *WorkspaceService) getAdmin(ctx context.Context, id string) (*model.WorkspaceAdmin, error) {
	list, err := s.workspaces.ListAdmin(ctx)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].ID == id {
			return &list[i], nil
		}
	}
	return nil, ErrWorkspaceNotFound
}

func (s *WorkspaceService) uniqueSlug(ctx context.Context, base string) (string, error) {
	for i := 0; i < 20; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", base, i+1)
		}
		exists, err := s.workspaces.SlugExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano()), nil
}

func slugFromDepartment(name string, id int64) string {
	s := workspaceSlugSanitizer.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	s = strings.Trim(s, "-")
	if s == "" || s == "-" {
		return "dept-" + strconv.FormatInt(id, 10)
	}
	if len(s) > 40 {
		s = s[:40]
	}
	return strings.Trim(s, "-")
}

func uniqueWorkspaceCount(desired map[string]map[string]string) int {
	return len(desired)
}

func desiredBitrixMemberships(
	users []model.BitrixUser,
	localIDs map[string]string,
	coverage map[int64]string,
	deps []model.BitrixDepartment,
) map[string]map[string]string {
	heads := map[int64]int64{}
	for _, d := range deps {
		if d.HeadBitrixID != nil && *d.HeadBitrixID > 0 {
			heads[d.BitrixID] = *d.HeadBitrixID
		}
	}
	out := map[string]map[string]string{}
	for _, u := range users {
		if !u.Active {
			continue
		}
		email := strings.ToLower(strings.TrimSpace(u.Email))
		localID, ok := localIDs[email]
		if !ok {
			continue
		}
		for _, deptID := range u.DepartmentIDs {
			wsID, ok := coverage[deptID]
			if !ok {
				continue
			}
			role := roleEmployee
			if heads[deptID] == u.BitrixID {
				role = roleDepartmentManager
			}
			if out[wsID] == nil {
				out[wsID] = map[string]string{}
			}
			if out[wsID][localID] != roleDepartmentManager {
				out[wsID][localID] = role
			}
		}
	}
	return out
}

func DepartmentCoverage(depts []model.BitrixDepartment, links []model.WorkspaceBitrixLink) map[int64]string {
	parent := map[int64]*int64{}
	for _, d := range depts {
		parent[d.BitrixID] = d.ParentBitrixID
	}
	explicit := map[int64]model.WorkspaceBitrixLink{}
	for _, l := range links {
		explicit[l.BitrixDepartmentID] = l
	}
	out := map[int64]string{}
	for _, d := range depts {
		if l, ok := explicit[d.BitrixID]; ok {
			out[d.BitrixID] = l.WorkspaceID
			continue
		}
		seen := map[int64]struct{}{}
		cur := d.ParentBitrixID
		for cur != nil {
			if _, loop := seen[*cur]; loop {
				break
			}
			seen[*cur] = struct{}{}
			if l, ok := explicit[*cur]; ok && l.IncludeDescendants {
				out[d.BitrixID] = l.WorkspaceID
				break
			}
			cur = parent[*cur]
		}
	}
	return out
}
