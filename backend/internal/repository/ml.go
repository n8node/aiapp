package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/n8node/aiapp/internal/model"
)

type MLModelRepository struct {
	pool *pgxpool.Pool
}

func NewMLModelRepository(pool *pgxpool.Pool) *MLModelRepository {
	return &MLModelRepository{pool: pool}
}

const mlModelCols = `
	id, slug, display_name, purpose, source_type, source_ref, status, artifact_hash,
	license, architecture, quantization, dimensions, context_length, gpu_device,
	gateway_alias, notes, created_by_user_id, approved_by_user_id, approved_at, created_at, updated_at`

func scanMLModel(row pgx.Row) (*model.MLModel, error) {
	var m model.MLModel
	err := row.Scan(
		&m.ID, &m.Slug, &m.DisplayName, &m.Purpose, &m.SourceType, &m.SourceRef, &m.Status, &m.ArtifactHash,
		&m.License, &m.Architecture, &m.Quantization, &m.Dimensions, &m.ContextLength, &m.GPUDevice,
		&m.GatewayAlias, &m.Notes, &m.CreatedByUserID, &m.ApprovedByUserID, &m.ApprovedAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &m, err
}

func (r *MLModelRepository) List(ctx context.Context) ([]model.MLModel, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+mlModelCols+` FROM ml_models ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.MLModel
	for rows.Next() {
		m, err := scanMLModel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	if out == nil {
		out = []model.MLModel{}
	}
	return out, rows.Err()
}

func (r *MLModelRepository) ListDeployed(ctx context.Context, purposes []string) ([]model.MLModel, error) {
	if len(purposes) == 0 {
		return []model.MLModel{}, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT `+mlModelCols+` FROM ml_models WHERE status = 'deployed' AND purpose = ANY($1) ORDER BY display_name`, purposes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.MLModel
	for rows.Next() {
		m, err := scanMLModel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	if out == nil {
		out = []model.MLModel{}
	}
	return out, rows.Err()
}

func (r *MLModelRepository) Get(ctx context.Context, id string) (*model.MLModel, error) {
	return scanMLModel(r.pool.QueryRow(ctx, `SELECT `+mlModelCols+` FROM ml_models WHERE id = $1`, id))
}

func (r *MLModelRepository) GetBySlug(ctx context.Context, slug string) (*model.MLModel, error) {
	return scanMLModel(r.pool.QueryRow(ctx, `SELECT `+mlModelCols+` FROM ml_models WHERE slug = $1`, slug))
}

func (r *MLModelRepository) DeployedByPurpose(ctx context.Context, purpose string) (*model.MLModel, error) {
	return scanMLModel(r.pool.QueryRow(ctx, `
		SELECT `+mlModelCols+` FROM ml_models WHERE purpose = $1 AND status = 'deployed'
		ORDER BY updated_at DESC LIMIT 1`, purpose))
}

func (r *MLModelRepository) Create(ctx context.Context, m *model.MLModel) (*model.MLModel, error) {
	return scanMLModel(r.pool.QueryRow(ctx, `
		INSERT INTO ml_models (
			slug, display_name, purpose, source_type, source_ref, status, license, architecture,
			quantization, dimensions, context_length, gpu_device, notes, created_by_user_id
		) VALUES ($1,$2,$3,$4,$5,'quarantine',$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING `+mlModelCols,
		m.Slug, m.DisplayName, m.Purpose, m.SourceType, m.SourceRef, m.License, m.Architecture,
		m.Quantization, m.Dimensions, m.ContextLength, m.GPUDevice, m.Notes, m.CreatedByUserID))
}

func (r *MLModelRepository) SetStatus(ctx context.Context, id, from, to, actorID string) error {
	var approved *string
	if to == model.ModelStatusApproved || to == model.ModelStatusDeployed {
		if actorID != "" {
			approved = &actorID
		}
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE ml_models SET status = $3, approved_by_user_id = COALESCE($4, approved_by_user_id),
			approved_at = CASE WHEN $3 IN ('approved','deployed') THEN NOW() ELSE approved_at END,
			updated_at = NOW()
		WHERE id = $1 AND status = $2`, id, from, to, approved)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *MLModelRepository) UndeployPurpose(ctx context.Context, purpose, exceptID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE ml_models SET status = 'approved', gateway_alias = NULL, updated_at = NOW()
		WHERE purpose = $1 AND status = 'deployed' AND id <> $2`, purpose, exceptID)
	return err
}

func (r *MLModelRepository) SetDeployed(ctx context.Context, id, alias string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE ml_models SET status = 'deployed', gateway_alias = $2, updated_at = NOW()
		WHERE id = $1 AND status IN ('approved','deployed')`, id, alias)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
