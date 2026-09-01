package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/n8node/aiapp/internal/model"
)

type TrainingRepository struct {
	pool *pgxpool.Pool
}

func NewTrainingRepository(pool *pgxpool.Pool) *TrainingRepository {
	return &TrainingRepository{pool: pool}
}

const trainingCols = `t.id, t.workspace_id, t.requested_by_user_id, u.email,
t.title, t.purpose, t.dataset_note, t.base_model, t.status, t.review_note,
t.reviewed_by_user_id, t.reviewed_at, t.created_at, t.updated_at`

func scanTraining(row interface{ Scan(dest ...any) error }) (*model.TrainingRequest, error) {
	var t model.TrainingRequest
	err := row.Scan(
		&t.ID, &t.WorkspaceID, &t.RequestedByUserID, &t.RequesterEmail,
		&t.Title, &t.Purpose, &t.DatasetNote, &t.BaseModel, &t.Status, &t.ReviewNote,
		&t.ReviewedByUserID, &t.ReviewedAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TrainingRepository) Create(ctx context.Context, t *model.TrainingRequest) (*model.TrainingRequest, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO training_requests (
			workspace_id, requested_by_user_id, title, purpose, dataset_note, base_model, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`,
		t.WorkspaceID, t.RequestedByUserID, t.Title, t.Purpose, t.DatasetNote, t.BaseModel, t.Status)
	if err := row.Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	return r.Get(ctx, t.ID)
}

func (r *TrainingRepository) Get(ctx context.Context, id string) (*model.TrainingRequest, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+trainingCols+`
		FROM training_requests t
		JOIN users u ON u.id = t.requested_by_user_id
		WHERE t.id = $1`, id)
	return scanTraining(row)
}

func (r *TrainingRepository) ListWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]model.TrainingRequest, int, error) {
	return r.list(ctx, "t.workspace_id = $1", []any{workspaceID}, limit, offset)
}

func (r *TrainingRepository) ListAll(ctx context.Context, status string, limit, offset int) ([]model.TrainingRequest, int, error) {
	where := "1=1"
	args := []any{}
	if strings.TrimSpace(status) != "" {
		where = "t.status = $1"
		args = append(args, strings.TrimSpace(status))
	}
	return r.list(ctx, where, args, limit, offset)
}

func (r *TrainingRepository) list(ctx context.Context, where string, args []any, limit, offset int) ([]model.TrainingRequest, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM training_requests t WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	n := len(args)
	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, `
		SELECT `+trainingCols+`
		FROM training_requests t
		JOIN users u ON u.id = t.requested_by_user_id
		WHERE `+where+`
		ORDER BY t.created_at DESC
		LIMIT $`+fmt.Sprint(n+1)+` OFFSET $`+fmt.Sprint(n+2), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []model.TrainingRequest
	for rows.Next() {
		item, err := scanTraining(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *item)
	}
	return out, total, rows.Err()
}

func (r *TrainingRepository) UpdateStatus(ctx context.Context, id, status, reviewNote string, reviewerID *string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE training_requests
		SET status = $2,
		    review_note = $3,
		    reviewed_by_user_id = $4,
		    reviewed_at = CASE WHEN $4 IS NULL THEN reviewed_at ELSE NOW() END,
		    updated_at = NOW()
		WHERE id = $1`, id, status, reviewNote, reviewerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
