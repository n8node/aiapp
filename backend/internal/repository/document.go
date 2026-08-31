package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/n8node/aiapp/internal/model"
)

type DocumentRepository struct {
	pool *pgxpool.Pool
}

func NewDocumentRepository(pool *pgxpool.Pool) *DocumentRepository {
	return &DocumentRepository{pool: pool}
}

const documentCols = `id, workspace_id, title, created_by_user_id, current_version_id, created_at, updated_at`

const versionCols = `
	id, document_id, workspace_id, version_n, disk_file_id, s3_key, original_name, mime_type, size,
	content_hash, status, engine, page_count, char_count, confidence, warnings, extracted_text,
	error_code, reviewed_by_user_id, reviewed_at, review_note, created_at, updated_at`

func scanDocument(row pgx.Row) (*model.Document, error) {
	var d model.Document
	err := row.Scan(&d.ID, &d.WorkspaceID, &d.Title, &d.CreatedByUserID, &d.CurrentVersionID, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &d, err
}

func scanVersion(row pgx.Row, includeText bool) (*model.DocumentVersion, error) {
	var v model.DocumentVersion
	var warnings []byte
	var extracted *string
	err := row.Scan(
		&v.ID, &v.DocumentID, &v.WorkspaceID, &v.VersionN, &v.DiskFileID, &v.S3Key, &v.OriginalName, &v.MimeType, &v.Size,
		&v.ContentHash, &v.Status, &v.Engine, &v.PageCount, &v.CharCount, &v.Confidence, &warnings, &extracted,
		&v.ErrorCode, &v.ReviewedByUserID, &v.ReviewedAt, &v.ReviewNote, &v.CreatedAt, &v.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(warnings) > 0 {
		_ = json.Unmarshal(warnings, &v.Warnings)
	}
	if v.Warnings == nil {
		v.Warnings = []string{}
	}
	if includeText {
		v.ExtractedText = extracted
	}
	return &v, nil
}

func (r *DocumentRepository) Create(ctx context.Context, d *model.Document) (*model.Document, error) {
	return scanDocument(r.pool.QueryRow(ctx, `
		INSERT INTO documents (workspace_id, title, created_by_user_id)
		VALUES ($1, $2, $3)
		RETURNING `+documentCols, d.WorkspaceID, d.Title, d.CreatedByUserID))
}

func (r *DocumentRepository) Get(ctx context.Context, workspaceID, id string) (*model.Document, error) {
	return scanDocument(r.pool.QueryRow(ctx, `
		SELECT `+documentCols+` FROM documents WHERE id = $1 AND workspace_id = $2`, id, workspaceID))
}

func (r *DocumentRepository) List(ctx context.Context, workspaceID string, limit int) ([]model.Document, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+documentCols+` FROM documents WHERE workspace_id = $1
		ORDER BY created_at DESC LIMIT $2`, workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Document
	for rows.Next() {
		d, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	if out == nil {
		out = []model.Document{}
	}
	return out, rows.Err()
}

func (r *DocumentRepository) SetCurrentVersion(ctx context.Context, workspaceID, documentID, versionID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE documents SET current_version_id = $3, updated_at = NOW()
		WHERE id = $1 AND workspace_id = $2`, documentID, workspaceID, versionID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *DocumentRepository) CreateVersion(ctx context.Context, v *model.DocumentVersion) (*model.DocumentVersion, error) {
	warn, _ := json.Marshal(v.Warnings)
	if v.Warnings == nil {
		warn = []byte("[]")
	}
	return scanVersion(r.pool.QueryRow(ctx, `
		INSERT INTO document_versions (
			document_id, workspace_id, version_n, disk_file_id, s3_key, original_name, mime_type, size,
			content_hash, status, warnings
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING `+versionCols,
		v.DocumentID, v.WorkspaceID, v.VersionN, v.DiskFileID, v.S3Key, v.OriginalName, v.MimeType, v.Size,
		v.ContentHash, v.Status, warn,
	), false)
}

func (r *DocumentRepository) GetVersion(ctx context.Context, workspaceID, versionID string, includeText bool) (*model.DocumentVersion, error) {
	return scanVersion(r.pool.QueryRow(ctx, `
		SELECT `+versionCols+` FROM document_versions WHERE id = $1 AND workspace_id = $2`, versionID, workspaceID), includeText)
}

func (r *DocumentRepository) NextVersionN(ctx context.Context, documentID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(version_n), 0) + 1 FROM document_versions WHERE document_id = $1`, documentID).Scan(&n)
	return n, err
}

func (r *DocumentRepository) FindByDiskFile(ctx context.Context, workspaceID, fileID string) (*model.DocumentVersion, error) {
	return scanVersion(r.pool.QueryRow(ctx, `
		SELECT `+versionCols+` FROM document_versions
		WHERE workspace_id = $1 AND disk_file_id = $2
		ORDER BY version_n DESC LIMIT 1`, workspaceID, fileID), false)
}

func (r *DocumentRepository) FindByHash(ctx context.Context, workspaceID, hash string) (*model.DocumentVersion, error) {
	if hash == "" {
		return nil, ErrNotFound
	}
	return scanVersion(r.pool.QueryRow(ctx, `
		SELECT `+versionCols+` FROM document_versions
		WHERE workspace_id = $1 AND content_hash = $2 AND status IN ('awaiting_review', 'published', 'queued', 'running')
		ORDER BY created_at DESC LIMIT 1`, workspaceID, hash), false)
}

func (r *DocumentRepository) UpdateVersionExtract(ctx context.Context, workspaceID, versionID, status, engine string, pageCount, charCount int, confidence float64, warnings []string, text, errorCode string) error {
	warn, _ := json.Marshal(warnings)
	if warnings == nil {
		warn = []byte("[]")
	}
	var errCode *string
	if errorCode != "" {
		errCode = &errorCode
	}
	var extracted *string
	if text != "" {
		extracted = &text
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE document_versions SET
			status = $3, engine = $4, page_count = $5, char_count = $6, confidence = $7,
			warnings = $8, extracted_text = $9, error_code = $10, updated_at = NOW()
		WHERE id = $1 AND workspace_id = $2 AND status IN ('queued', 'running')`,
		versionID, workspaceID, status, engine, pageCount, charCount, confidence, warn, extracted, errCode)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *DocumentRepository) UpdateVersionStatus(ctx context.Context, workspaceID, versionID, fromStatus, toStatus, reviewerID, note string) error {
	var rev *string
	if reviewerID != "" {
		rev = &reviewerID
	}
	var n *string
	if note != "" {
		n = &note
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE document_versions SET
			status = $4, reviewed_by_user_id = $5, reviewed_at = NOW(), review_note = $6, updated_at = NOW()
		WHERE id = $1 AND workspace_id = $2 AND status = $3`,
		versionID, workspaceID, fromStatus, toStatus, rev, n)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *DocumentRepository) SetVersionStatus(ctx context.Context, workspaceID, versionID, fromStatus, toStatus string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE document_versions SET status = $4, updated_at = NOW()
		WHERE id = $1 AND workspace_id = $2 AND status = $3`,
		versionID, workspaceID, fromStatus, toStatus)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *DocumentRepository) ResetVersionForRetry(ctx context.Context, workspaceID, versionID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE document_versions SET status = 'queued', error_code = NULL, engine = NULL,
			extracted_text = NULL, updated_at = NOW()
		WHERE id = $1 AND workspace_id = $2 AND status IN ('failed', 'rejected')`, versionID, workspaceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *DocumentRepository) EnqueueJob(ctx context.Context, j *model.IngestJob) (*model.IngestJob, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO ingest_jobs (workspace_id, document_version_id, created_by_user_id, status)
		VALUES ($1, $2, $3, 'queued')
		RETURNING id, workspace_id, document_version_id, created_by_user_id, status, attempts, max_attempts, lease_until, lease_owner, last_error`,
		j.WorkspaceID, j.DocumentVersionID, j.CreatedByUserID)
	return scanJob(row)
}

func (r *DocumentRepository) ClaimJob(ctx context.Context, owner string, lease time.Duration) (*model.IngestJob, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE ingest_jobs SET status = 'running', attempts = attempts + 1, lease_owner = $1,
			lease_until = NOW() + ($2 * INTERVAL '1 second'), updated_at = NOW()
		WHERE id = (
			SELECT id FROM ingest_jobs
			WHERE status = 'queued'
			   OR (status = 'running' AND lease_until IS NOT NULL AND lease_until < NOW() AND attempts < max_attempts)
			ORDER BY created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, workspace_id, document_version_id, created_by_user_id, status, attempts, max_attempts, lease_until, lease_owner, last_error`,
		owner, int(lease.Seconds()))
	j, err := scanJob(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return j, err
}

func (r *DocumentRepository) FinishJob(ctx context.Context, id, status, lastError string) error {
	var errCode *string
	if lastError != "" {
		errCode = &lastError
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE ingest_jobs SET status = $2, last_error = $3, lease_until = NULL, updated_at = NOW()
		WHERE id = $1`, id, status, errCode)
	return err
}

func scanJob(row pgx.Row) (*model.IngestJob, error) {
	var j model.IngestJob
	err := row.Scan(&j.ID, &j.WorkspaceID, &j.DocumentVersionID, &j.CreatedByUserID, &j.Status, &j.Attempts, &j.MaxAttempts, &j.LeaseUntil, &j.LeaseOwner, &j.LastError)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &j, err
}
