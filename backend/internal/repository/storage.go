package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/n8node/aiapp/internal/model"
)

type StorageSettingsRepository struct {
	pool *pgxpool.Pool
}

func NewStorageSettingsRepository(pool *pgxpool.Pool) *StorageSettingsRepository {
	return &StorageSettingsRepository{pool: pool}
}

func (r *StorageSettingsRepository) Get(ctx context.Context) (*model.StorageSettingsRecord, error) {
	var raw []byte
	var rec model.StorageSettingsRecord
	err := r.pool.QueryRow(ctx, `
		SELECT config, updated_at FROM storage_settings WHERE id = 1`).Scan(&raw, &rec.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &rec.Config); err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *StorageSettingsRepository) Update(ctx context.Context, config model.StorageSettings) (*model.StorageSettingsRecord, error) {
	raw, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	var out []byte
	var rec model.StorageSettingsRecord
	err = r.pool.QueryRow(ctx, `
		UPDATE storage_settings
		SET config = $1, updated_at = NOW()
		WHERE id = 1
		RETURNING config, updated_at`, raw).Scan(&out, &rec.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(out, &rec.Config); err != nil {
		return nil, err
	}
	return &rec, nil
}
