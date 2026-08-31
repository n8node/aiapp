package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/n8node/aiapp/internal/model"
)

type WorkspaceMember struct {
	WorkspaceID string
	UserID      string
	Role        string
	Source      string
}

func (r *WorkspaceRepository) GetByID(ctx context.Context, id string) (*model.Workspace, error) {
	var w model.Workspace
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, slug, owner_id, created_at
		FROM workspaces WHERE id = $1`, id).Scan(&w.ID, &w.Name, &w.Slug, &w.OwnerID, &w.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &w, nil
}

func (r *WorkspaceRepository) ListAdmin(ctx context.Context) ([]model.WorkspaceAdmin, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT w.id, w.name, w.slug, w.owner_id, w.created_at,
			(SELECT COUNT(*) FROM workspace_members m WHERE m.workspace_id = w.id)
		FROM workspaces w
		ORDER BY w.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.WorkspaceAdmin, 0)
	index := map[string]int{}
	for rows.Next() {
		var item model.WorkspaceAdmin
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.OwnerID, &item.CreatedAt, &item.MemberCount); err != nil {
			return nil, err
		}
		item.BitrixDepartments = []model.WorkspaceBitrixLink{}
		index[item.ID] = len(out)
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	links, err := r.ListLinks(ctx)
	if err != nil {
		return nil, err
	}
	for _, link := range links {
		i, ok := index[link.WorkspaceID]
		if !ok {
			continue
		}
		out[i].BitrixDepartments = append(out[i].BitrixDepartments, link)
	}
	return out, nil
}

func (r *WorkspaceRepository) ListLinks(ctx context.Context) ([]model.WorkspaceBitrixLink, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT l.workspace_id, w.name, l.bitrix_department_id, COALESCE(d.name, ''), l.include_descendants
		FROM workspace_bitrix_departments l
		JOIN workspaces w ON w.id = l.workspace_id
		LEFT JOIN bitrix_departments d ON d.bitrix_id = l.bitrix_department_id
		ORDER BY w.name, d.sort, d.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.WorkspaceBitrixLink, 0)
	for rows.Next() {
		var l model.WorkspaceBitrixLink
		if err := rows.Scan(&l.WorkspaceID, &l.WorkspaceName, &l.BitrixDepartmentID, &l.DepartmentName, &l.IncludeDescendants); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *WorkspaceRepository) LinkDepartment(ctx context.Context, workspaceID string, deptID int64, includeDescendants bool) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO workspace_bitrix_departments (workspace_id, bitrix_department_id, include_descendants)
		VALUES ($1, $2, $3)
		ON CONFLICT (bitrix_department_id) DO UPDATE
		SET workspace_id = EXCLUDED.workspace_id,
		    include_descendants = EXCLUDED.include_descendants`,
		workspaceID, deptID, includeDescendants)
	return err
}

func (r *WorkspaceRepository) UnlinkDepartment(ctx context.Context, workspaceID string, deptID int64) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM workspace_bitrix_departments
		WHERE workspace_id = $1 AND bitrix_department_id = $2`, workspaceID, deptID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *WorkspaceRepository) ListBitrixMembers(ctx context.Context) ([]WorkspaceMember, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT workspace_id, user_id, role, source
		FROM workspace_members
		WHERE source = 'bitrix'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]WorkspaceMember, 0)
	for rows.Next() {
		var m WorkspaceMember
		if err := rows.Scan(&m.WorkspaceID, &m.UserID, &m.Role, &m.Source); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *WorkspaceRepository) GetMember(ctx context.Context, workspaceID, userID string) (*WorkspaceMember, error) {
	var m WorkspaceMember
	err := r.pool.QueryRow(ctx, `
		SELECT workspace_id, user_id, role, source
		FROM workspace_members
		WHERE workspace_id = $1 AND user_id = $2`, workspaceID, userID).Scan(&m.WorkspaceID, &m.UserID, &m.Role, &m.Source)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (r *WorkspaceRepository) UpsertBitrixMember(ctx context.Context, workspaceID, userID, role string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role, source)
		VALUES ($1, $2, $3, 'bitrix')
		ON CONFLICT (workspace_id, user_id) DO UPDATE
		SET role = EXCLUDED.role
		WHERE workspace_members.source = 'bitrix'`, workspaceID, userID, role)
	return err
}

func (r *WorkspaceRepository) DeleteBitrixMember(ctx context.Context, workspaceID, userID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM workspace_members
		WHERE workspace_id = $1 AND user_id = $2 AND source = 'bitrix'`, workspaceID, userID)
	return err
}
