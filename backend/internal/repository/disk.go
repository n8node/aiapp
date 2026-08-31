package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/n8node/aiapp/internal/model"
)

type DiskRepository struct {
	pool *pgxpool.Pool
}

func NewDiskRepository(pool *pgxpool.Pool) *DiskRepository {
	return &DiskRepository{pool: pool}
}

const diskFileCols = `
	id, workspace_id, folder_id, created_by_user_id, name, mime_type, size, s3_key,
	deleted_at, trash_batch_id, created_at, updated_at`

func scanDiskFile(row pgx.Row) (*model.DiskFile, error) {
	var f model.DiskFile
	err := row.Scan(
		&f.ID, &f.WorkspaceID, &f.FolderID, &f.CreatedByUserID, &f.Name, &f.MimeType, &f.Size, &f.S3Key,
		&f.DeletedAt, &f.TrashBatchID, &f.CreatedAt, &f.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &f, err
}

const diskFolderCols = `
	id, workspace_id, parent_id, name, created_by_user_id, deleted_at, trash_batch_id, created_at, updated_at`

func scanDiskFolder(row pgx.Row) (*model.DiskFolder, error) {
	var f model.DiskFolder
	err := row.Scan(
		&f.ID, &f.WorkspaceID, &f.ParentID, &f.Name, &f.CreatedByUserID, &f.DeletedAt, &f.TrashBatchID, &f.CreatedAt, &f.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &f, err
}

func (r *DiskRepository) GetFile(ctx context.Context, workspaceID, fileID string, includeDeleted bool) (*model.DiskFile, error) {
	q := `SELECT ` + diskFileCols + ` FROM workspace_files WHERE id = $1 AND workspace_id = $2`
	if !includeDeleted {
		q += ` AND deleted_at IS NULL`
	}
	return scanDiskFile(r.pool.QueryRow(ctx, q, fileID, workspaceID))
}

func (r *DiskRepository) GetFileByS3Key(ctx context.Context, workspaceID, s3Key string) (*model.DiskFile, error) {
	return scanDiskFile(r.pool.QueryRow(ctx,
		`SELECT `+diskFileCols+` FROM workspace_files WHERE workspace_id = $1 AND s3_key = $2`,
		workspaceID, s3Key,
	))
}

type DiskFileFilter struct {
	WorkspaceID string
	FolderID    *string
	ScopeAll    bool
	TypeFilter  string
	RecentOnly  bool
	Limit       int
}

func (r *DiskRepository) ListFiles(ctx context.Context, filter DiskFileFilter) ([]model.DiskFile, error) {
	q := `SELECT ` + diskFileCols + ` FROM workspace_files WHERE workspace_id = $1 AND deleted_at IS NULL`
	args := []any{filter.WorkspaceID}
	n := 2
	if !filter.ScopeAll {
		if filter.FolderID != nil && *filter.FolderID != "" {
			q += fmt.Sprintf(` AND folder_id = $%d`, n)
			args = append(args, *filter.FolderID)
			n++
		} else {
			q += ` AND folder_id IS NULL`
		}
	}
	switch filter.TypeFilter {
	case "image":
		q += ` AND mime_type LIKE 'image/%'`
	case "video":
		q += ` AND mime_type LIKE 'video/%'`
	}
	q += ` ORDER BY created_at DESC`
	if filter.RecentOnly {
		if filter.Limit <= 0 {
			filter.Limit = 100
		}
	}
	if filter.Limit > 0 {
		q += fmt.Sprintf(` LIMIT $%d`, n)
		args = append(args, filter.Limit)
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.DiskFile
	for rows.Next() {
		f, err := scanDiskFile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

func (r *DiskRepository) CreateFile(ctx context.Context, f *model.DiskFile) (*model.DiskFile, error) {
	return scanDiskFile(r.pool.QueryRow(ctx, `
		INSERT INTO workspace_files (workspace_id, folder_id, created_by_user_id, name, mime_type, size, s3_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+diskFileCols, f.WorkspaceID, f.FolderID, f.CreatedByUserID, f.Name, f.MimeType, f.Size, f.S3Key))
}

func (r *DiskRepository) UpdateFileName(ctx context.Context, workspaceID, fileID, name string) (*model.DiskFile, error) {
	return scanDiskFile(r.pool.QueryRow(ctx, `
		UPDATE workspace_files SET name = $3, updated_at = NOW()
		WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
		RETURNING `+diskFileCols, fileID, workspaceID, name))
}

func (r *DiskRepository) UpdateFileFolder(ctx context.Context, workspaceID, fileID string, folderID *string) (*model.DiskFile, error) {
	return scanDiskFile(r.pool.QueryRow(ctx, `
		UPDATE workspace_files SET folder_id = $3, updated_at = NOW()
		WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
		RETURNING `+diskFileCols, fileID, workspaceID, folderID))
}

func (r *DiskRepository) SoftDeleteFiles(ctx context.Context, workspaceID string, ids []string, batchID string, at time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE workspace_files SET deleted_at = $3, trash_batch_id = $4, updated_at = NOW()
		WHERE workspace_id = $1 AND deleted_at IS NULL AND id = ANY($2::uuid[])`,
		workspaceID, ids, at, batchID)
	return err
}

func (r *DiskRepository) RestoreFile(ctx context.Context, workspaceID, fileID string, folderID *string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE workspace_files SET deleted_at = NULL, trash_batch_id = NULL, folder_id = $3, updated_at = NOW()
		WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NOT NULL`,
		fileID, workspaceID, folderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *DiskRepository) RestoreFilesByBatch(ctx context.Context, workspaceID, batchID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE workspace_files SET deleted_at = NULL, trash_batch_id = NULL, updated_at = NOW()
		WHERE workspace_id = $1 AND trash_batch_id = $2`, workspaceID, batchID)
	return err
}

func (r *DiskRepository) DeleteFilePermanent(ctx context.Context, workspaceID, fileID string) (*model.DiskFile, error) {
	return scanDiskFile(r.pool.QueryRow(ctx, `
		DELETE FROM workspace_files WHERE id = $1 AND workspace_id = $2
		RETURNING `+diskFileCols, fileID, workspaceID))
}

func (r *DiskRepository) ListTrashedFilesTop(ctx context.Context, workspaceID string) ([]model.DiskFile, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+diskFileCols+` FROM workspace_files f
		WHERE f.workspace_id = $1 AND f.deleted_at IS NOT NULL
		  AND (
		    f.folder_id IS NULL
		    OR NOT EXISTS (
		      SELECT 1 FROM workspace_folders p
		      WHERE p.id = f.folder_id AND p.workspace_id = f.workspace_id AND p.deleted_at IS NOT NULL
		    )
		  )
		ORDER BY f.deleted_at DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.DiskFile
	for rows.Next() {
		f, err := scanDiskFile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

func (r *DiskRepository) CollectFileIDsInFolder(ctx context.Context, folderID string, activeOnly bool) ([]string, error) {
	q := `SELECT id FROM workspace_files WHERE folder_id = $1`
	if activeOnly {
		q += ` AND deleted_at IS NULL`
	}
	rows, err := r.pool.Query(ctx, q, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *DiskRepository) ListFilesByIDs(ctx context.Context, workspaceID string, ids []string, includeDeleted bool) ([]model.DiskFile, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	q := `SELECT ` + diskFileCols + ` FROM workspace_files WHERE workspace_id = $1 AND id = ANY($2::uuid[])`
	if !includeDeleted {
		q += ` AND deleted_at IS NULL`
	}
	rows, err := r.pool.Query(ctx, q, workspaceID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.DiskFile
	for rows.Next() {
		f, err := scanDiskFile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

func (r *DiskRepository) ListAllTrashedFiles(ctx context.Context, workspaceID string) ([]model.DiskFile, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+diskFileCols+` FROM workspace_files WHERE workspace_id = $1 AND deleted_at IS NOT NULL`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.DiskFile
	for rows.Next() {
		f, err := scanDiskFile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

func (r *DiskRepository) GetFolder(ctx context.Context, workspaceID, folderID string, includeDeleted bool) (*model.DiskFolder, error) {
	q := `SELECT ` + diskFolderCols + ` FROM workspace_folders WHERE id = $1 AND workspace_id = $2`
	if !includeDeleted {
		q += ` AND deleted_at IS NULL`
	}
	return scanDiskFolder(r.pool.QueryRow(ctx, q, folderID, workspaceID))
}

func (r *DiskRepository) FolderExistsActive(ctx context.Context, workspaceID string, folderID *string) error {
	if folderID == nil || *folderID == "" {
		return nil
	}
	_, err := r.GetFolder(ctx, workspaceID, *folderID, false)
	return err
}

func (r *DiskRepository) ListFolders(ctx context.Context, workspaceID string, parentID *string, scopeAll bool) ([]model.DiskFolder, error) {
	q := `SELECT ` + diskFolderCols + ` FROM workspace_folders WHERE workspace_id = $1 AND deleted_at IS NULL`
	args := []any{workspaceID}
	if !scopeAll {
		if parentID != nil && *parentID != "" {
			q += ` AND parent_id = $2`
			args = append(args, *parentID)
		} else {
			q += ` AND parent_id IS NULL`
		}
	}
	q += ` ORDER BY name ASC`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.DiskFolder
	for rows.Next() {
		f, err := scanDiskFolder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

func (r *DiskRepository) CreateFolder(ctx context.Context, f *model.DiskFolder) (*model.DiskFolder, error) {
	return scanDiskFolder(r.pool.QueryRow(ctx, `
		INSERT INTO workspace_folders (workspace_id, parent_id, name, created_by_user_id)
		VALUES ($1, $2, $3, $4)
		RETURNING `+diskFolderCols, f.WorkspaceID, f.ParentID, f.Name, f.CreatedByUserID))
}

func (r *DiskRepository) UpdateFolderName(ctx context.Context, workspaceID, folderID, name string) (*model.DiskFolder, error) {
	return scanDiskFolder(r.pool.QueryRow(ctx, `
		UPDATE workspace_folders SET name = $3, updated_at = NOW()
		WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
		RETURNING `+diskFolderCols, folderID, workspaceID, name))
}

func (r *DiskRepository) UpdateFolderParent(ctx context.Context, workspaceID, folderID string, parentID *string) (*model.DiskFolder, error) {
	return scanDiskFolder(r.pool.QueryRow(ctx, `
		UPDATE workspace_folders SET parent_id = $3, updated_at = NOW()
		WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
		RETURNING `+diskFolderCols, folderID, workspaceID, parentID))
}

func (r *DiskRepository) SoftDeleteFolders(ctx context.Context, workspaceID string, ids []string, batchID string, at time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE workspace_folders SET deleted_at = $3, trash_batch_id = $4, updated_at = NOW()
		WHERE workspace_id = $1 AND deleted_at IS NULL AND id = ANY($2::uuid[])`,
		workspaceID, ids, at, batchID)
	return err
}

func (r *DiskRepository) RestoreFolder(ctx context.Context, workspaceID, folderID string, parentID *string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE workspace_folders SET deleted_at = NULL, trash_batch_id = NULL, parent_id = $3, updated_at = NOW()
		WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NOT NULL`,
		folderID, workspaceID, parentID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *DiskRepository) RestoreFoldersByBatch(ctx context.Context, workspaceID, batchID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE workspace_folders SET deleted_at = NULL, trash_batch_id = NULL, updated_at = NOW()
		WHERE workspace_id = $1 AND trash_batch_id = $2`, workspaceID, batchID)
	return err
}

func (r *DiskRepository) DeleteFolderPermanent(ctx context.Context, workspaceID, folderID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM workspace_folders WHERE id = $1 AND workspace_id = $2`, folderID, workspaceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *DiskRepository) DeleteFoldersByIDs(ctx context.Context, workspaceID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM workspace_folders WHERE workspace_id = $1 AND id = ANY($2::uuid[])`, workspaceID, ids)
	return err
}

func (r *DiskRepository) ListTrashedFoldersTop(ctx context.Context, workspaceID string) ([]model.DiskFolder, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+diskFolderCols+` FROM workspace_folders f
		WHERE f.workspace_id = $1 AND f.deleted_at IS NOT NULL
		  AND (
		    f.parent_id IS NULL
		    OR NOT EXISTS (
		      SELECT 1 FROM workspace_folders p
		      WHERE p.id = f.parent_id AND p.workspace_id = f.workspace_id AND p.deleted_at IS NOT NULL
		    )
		  )
		ORDER BY f.deleted_at DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.DiskFolder
	for rows.Next() {
		f, err := scanDiskFolder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

func (r *DiskRepository) CollectSubtreeIDs(ctx context.Context, folderID string, activeOnly bool) ([]string, error) {
	q := `
		WITH RECURSIVE tree AS (
			SELECT id FROM workspace_folders WHERE id = $1
			UNION ALL
			SELECT f.id FROM workspace_folders f JOIN tree t ON f.parent_id = t.id`
	if activeOnly {
		q += ` WHERE f.deleted_at IS NULL`
	}
	q += `) SELECT id FROM tree`
	rows, err := r.pool.Query(ctx, q, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *DiskRepository) IsDescendantOf(ctx context.Context, folderID, ancestorID string) (bool, error) {
	if folderID == ancestorID {
		return true, nil
	}
	var exists bool
	err := r.pool.QueryRow(ctx, `
		WITH RECURSIVE ancestors AS (
			SELECT id, parent_id FROM workspace_folders WHERE id = $1
			UNION ALL
			SELECT f.id, f.parent_id FROM workspace_folders f
			JOIN ancestors a ON f.id = a.parent_id
		)
		SELECT EXISTS(SELECT 1 FROM ancestors WHERE id = $2)`, folderID, ancestorID).Scan(&exists)
	return exists, err
}

func (r *DiskRepository) CountFilesInFolder(ctx context.Context, folderID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM workspace_files WHERE folder_id = $1 AND deleted_at IS NULL`, folderID).Scan(&n)
	return n, err
}

func (r *DiskRepository) Breadcrumbs(ctx context.Context, workspaceID, folderID string) ([]model.FolderBreadcrumb, error) {
	rows, err := r.pool.Query(ctx, `
		WITH RECURSIVE chain AS (
			SELECT id, parent_id, name FROM workspace_folders
			WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
			UNION ALL
			SELECT f.id, f.parent_id, f.name FROM workspace_folders f
			JOIN chain c ON f.id = c.parent_id
			WHERE f.deleted_at IS NULL
		)
		SELECT id, name FROM chain`, folderID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var crumbs []model.FolderBreadcrumb
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		idCopy := id
		crumbs = append(crumbs, model.FolderBreadcrumb{ID: &idCopy, Name: name})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i, j := 0, len(crumbs)-1; i < j; i, j = i+1, j-1 {
		crumbs[i], crumbs[j] = crumbs[j], crumbs[i]
	}
	return crumbs, nil
}

func (r *DiskRepository) SiblingNameTaken(ctx context.Context, workspaceID string, parentID *string, name, excludeID string) (bool, error) {
	q := `SELECT EXISTS(
		SELECT 1 FROM workspace_folders
		WHERE workspace_id = $1 AND deleted_at IS NULL AND lower(name) = lower($2)`
	args := []any{workspaceID, name}
	n := 3
	if parentID != nil && *parentID != "" {
		q += fmt.Sprintf(` AND parent_id = $%d`, n)
		args = append(args, *parentID)
		n++
	} else {
		q += ` AND parent_id IS NULL`
	}
	if strings.TrimSpace(excludeID) != "" {
		q += fmt.Sprintf(` AND id <> $%d`, n)
		args = append(args, excludeID)
	}
	q += `)`
	var taken bool
	err := r.pool.QueryRow(ctx, q, args...).Scan(&taken)
	return taken, err
}
