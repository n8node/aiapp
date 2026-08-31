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

type KnowledgeRepository struct {
	pool *pgxpool.Pool
}

func NewKnowledgeRepository(pool *pgxpool.Pool) *KnowledgeRepository {
	return &KnowledgeRepository{pool: pool}
}

const kbCols = `id, workspace_id, name, chunk_size, chunk_overlap, embedding_model_id, created_by_user_id, created_at, updated_at`

func scanKB(row pgx.Row) (*model.KnowledgeBase, error) {
	var k model.KnowledgeBase
	var createdBy *string
	err := row.Scan(&k.ID, &k.WorkspaceID, &k.Name, &k.ChunkSize, &k.ChunkOverlap, &k.EmbeddingModelID, &createdBy, &k.CreatedAt, &k.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &k, err
}

func (r *KnowledgeRepository) Create(ctx context.Context, k *model.KnowledgeBase, createdBy *string) (*model.KnowledgeBase, error) {
	return scanKB(r.pool.QueryRow(ctx, `
		INSERT INTO knowledge_bases (workspace_id, name, chunk_size, chunk_overlap, embedding_model_id, created_by_user_id)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING `+kbCols, k.WorkspaceID, k.Name, k.ChunkSize, k.ChunkOverlap, k.EmbeddingModelID, createdBy))
}

func (r *KnowledgeRepository) Get(ctx context.Context, workspaceID, id string) (*model.KnowledgeBase, error) {
	return scanKB(r.pool.QueryRow(ctx, `SELECT `+kbCols+` FROM knowledge_bases WHERE id = $1 AND workspace_id = $2`, id, workspaceID))
}

func (r *KnowledgeRepository) List(ctx context.Context, workspaceID string) ([]model.KnowledgeBase, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT k.id, k.workspace_id, k.name, k.chunk_size, k.chunk_overlap, k.embedding_model_id, k.created_by_user_id, k.created_at, k.updated_at,
			COUNT(f.file_id)::int,
			COUNT(f.file_id) FILTER (WHERE f.status = 'indexed')::int
		FROM knowledge_bases k
		LEFT JOIN knowledge_base_files f ON f.knowledge_base_id = k.id
		WHERE k.workspace_id = $1
		GROUP BY k.id
		ORDER BY k.created_at DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.KnowledgeBase
	for rows.Next() {
		var k model.KnowledgeBase
		var createdBy *string
		if err := rows.Scan(&k.ID, &k.WorkspaceID, &k.Name, &k.ChunkSize, &k.ChunkOverlap, &k.EmbeddingModelID, &createdBy, &k.CreatedAt, &k.UpdatedAt, &k.FileCount, &k.IndexedCount); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	if out == nil {
		out = []model.KnowledgeBase{}
	}
	return out, rows.Err()
}

func (r *KnowledgeRepository) Delete(ctx context.Context, workspaceID, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM knowledge_bases WHERE id = $1 AND workspace_id = $2`, id, workspaceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *KnowledgeRepository) Rename(ctx context.Context, workspaceID, id, name string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE knowledge_bases SET name = $3, updated_at = NOW() WHERE id = $1 AND workspace_id = $2`, id, workspaceID, name)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *KnowledgeRepository) AttachFiles(ctx context.Context, kbID string, fileIDs []string) error {
	for _, id := range fileIDs {
		_, err := r.pool.Exec(ctx, `
			INSERT INTO knowledge_base_files (knowledge_base_id, file_id, status)
			VALUES ($1,$2,'pending')
			ON CONFLICT (knowledge_base_id, file_id) DO NOTHING`, kbID, id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *KnowledgeRepository) ListFiles(ctx context.Context, kbID string) ([]model.KnowledgeBaseFile, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT f.file_id, COALESCE(wf.name, ''), f.content_hash, f.chunk_count, f.status, f.error_code
		FROM knowledge_base_files f
		LEFT JOIN workspace_files wf ON wf.id = f.file_id
		WHERE f.knowledge_base_id = $1
		ORDER BY wf.name`, kbID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.KnowledgeBaseFile
	for rows.Next() {
		var it model.KnowledgeBaseFile
		if err := rows.Scan(&it.FileID, &it.Name, &it.ContentHash, &it.ChunkCount, &it.Status, &it.ErrorCode); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	if out == nil {
		out = []model.KnowledgeBaseFile{}
	}
	return out, rows.Err()
}

func (r *KnowledgeRepository) SetFileStatus(ctx context.Context, kbID, fileID, status, hash string, chunks int, errCode string) error {
	var code *string
	if errCode != "" {
		code = &errCode
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE knowledge_base_files SET status = $3, content_hash = $4, chunk_count = $5, error_code = $6, updated_at = NOW()
		WHERE knowledge_base_id = $1 AND file_id = $2`, kbID, fileID, status, hash, chunks, code)
	return err
}

func (r *KnowledgeRepository) DeleteChunksForFile(ctx context.Context, kbID, fileID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM knowledge_chunks WHERE knowledge_base_id = $1 AND file_id = $2`, kbID, fileID)
	return err
}

func (r *KnowledgeRepository) InsertChunk(ctx context.Context, workspaceID, kbID, fileID string, index int, content, hash string, embedding []float64) error {
	vec := formatVector(embedding)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO knowledge_chunks (workspace_id, knowledge_base_id, file_id, chunk_index, content, content_hash, embedding, token_count)
		VALUES ($1,$2,$3,$4,$5,$6,$7::vector,$8)`,
		workspaceID, kbID, fileID, index, content, hash, vec, len([]rune(content)))
	return err
}

func (r *KnowledgeRepository) SearchVector(ctx context.Context, workspaceID, kbID string, embedding []float64, limit int, threshold float64) ([]model.KnowledgeHit, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	vec := formatVector(embedding)
	rows, err := r.pool.Query(ctx, `
		SELECT c.file_id, COALESCE(wf.name, ''), c.chunk_index, c.content,
			1 - (c.embedding <=> $3::vector) AS similarity
		FROM knowledge_chunks c
		LEFT JOIN workspace_files wf ON wf.id = c.file_id
		WHERE c.workspace_id = $1 AND c.knowledge_base_id = $2
			AND c.embedding IS NOT NULL
			AND 1 - (c.embedding <=> $3::vector) >= $4
		ORDER BY c.embedding <=> $3::vector
		LIMIT $5`, workspaceID, kbID, vec, threshold, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.KnowledgeHit
	for rows.Next() {
		var h model.KnowledgeHit
		if err := rows.Scan(&h.FileID, &h.FileName, &h.ChunkIndex, &h.Content, &h.Similarity); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	if out == nil {
		out = []model.KnowledgeHit{}
	}
	return out, rows.Err()
}

func (r *KnowledgeRepository) EnqueueJob(ctx context.Context, j *model.VectorizeJob) (*model.VectorizeJob, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO vectorize_jobs (workspace_id, knowledge_base_id, created_by_user_id, status, total)
		VALUES ($1,$2,$3,'queued',$4)
		RETURNING id, workspace_id, knowledge_base_id, created_by_user_id, status, attempts, max_attempts, lease_until, lease_owner, last_error, processed, total`,
		j.WorkspaceID, j.KnowledgeBaseID, j.CreatedByUserID, j.Total)
	return scanVecJob(row)
}

func (r *KnowledgeRepository) ClaimJob(ctx context.Context, owner string, lease time.Duration) (*model.VectorizeJob, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE vectorize_jobs SET status = 'running', attempts = attempts + 1, lease_owner = $1,
			lease_until = NOW() + ($2 * INTERVAL '1 second'), updated_at = NOW()
		WHERE id = (
			SELECT id FROM vectorize_jobs
			WHERE status = 'queued'
			   OR (status = 'running' AND lease_until IS NOT NULL AND lease_until < NOW() AND attempts < max_attempts)
			ORDER BY created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, workspace_id, knowledge_base_id, created_by_user_id, status, attempts, max_attempts, lease_until, lease_owner, last_error, processed, total`,
		owner, int(lease.Seconds()))
	return scanVecJob(row)
}

func (r *KnowledgeRepository) FinishJob(ctx context.Context, id, status, lastError string, processed, total int) error {
	var errCode *string
	if lastError != "" {
		errCode = &lastError
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE vectorize_jobs SET status = $2, last_error = $3, processed = $4, total = $5, lease_until = NULL, updated_at = NOW()
		WHERE id = $1`, id, status, errCode, processed, total)
	return err
}

func (r *KnowledgeRepository) TouchJob(ctx context.Context, id string, processed, total int) error {
	_, err := r.pool.Exec(ctx, `UPDATE vectorize_jobs SET processed = $2, total = $3, updated_at = NOW() WHERE id = $1`, id, processed, total)
	return err
}

func (r *KnowledgeRepository) RunningJobForKB(ctx context.Context, kbID string) (*model.VectorizeJob, error) {
	return scanVecJob(r.pool.QueryRow(ctx, `
		SELECT id, workspace_id, knowledge_base_id, created_by_user_id, status, attempts, max_attempts, lease_until, lease_owner, last_error, processed, total
		FROM vectorize_jobs
		WHERE knowledge_base_id = $1 AND status IN ('queued','running')
		ORDER BY created_at DESC LIMIT 1`, kbID))
}

func scanVecJob(row pgx.Row) (*model.VectorizeJob, error) {
	var j model.VectorizeJob
	err := row.Scan(&j.ID, &j.WorkspaceID, &j.KnowledgeBaseID, &j.CreatedByUserID, &j.Status, &j.Attempts, &j.MaxAttempts, &j.LeaseUntil, &j.LeaseOwner, &j.LastError, &j.Processed, &j.Total)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &j, err
}

func formatVector(v []float64) string {
	if len(v) == 0 {
		return "[]"
	}
	var b strings.Builder
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(fmt.Sprintf("%g", x))
	}
	b.WriteByte(']')
	return b.String()
}
