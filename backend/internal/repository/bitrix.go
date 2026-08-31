package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/n8node/aiapp/internal/bitrix"
	"github.com/n8node/aiapp/internal/model"
)

type BitrixRepository struct {
	pool *pgxpool.Pool
}

func NewBitrixRepository(pool *pgxpool.Pool) *BitrixRepository {
	return &BitrixRepository{pool: pool}
}

func (r *BitrixRepository) ReplaceSnapshot(ctx context.Context, deps []bitrix.Department, users []bitrix.User) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM bitrix_users`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bitrix_departments`); err != nil {
		return err
	}
	for _, d := range deps {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bitrix_departments (bitrix_id, parent_bitrix_id, name, sort, synced_at)
			VALUES ($1, $2, $3, $4, NOW())`, d.ID, d.ParentID, d.Name, d.Sort); err != nil {
			return err
		}
	}
	for _, u := range users {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bitrix_users (bitrix_id, email, name, last_name, active, department_ids, synced_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
			u.ID, u.Email, u.Name, u.LastName, u.Active, u.DepartmentIDs); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *BitrixRepository) Counts(ctx context.Context) (deps, users int, err error) {
	err = r.pool.QueryRow(ctx, `SELECT
		(SELECT COUNT(*) FROM bitrix_departments),
		(SELECT COUNT(*) FROM bitrix_users)`).Scan(&deps, &users)
	return
}

func (r *BitrixRepository) ListDepartments(ctx context.Context) ([]model.BitrixDepartment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT bitrix_id, parent_bitrix_id, name, sort
		FROM bitrix_departments
		ORDER BY sort, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.BitrixDepartment, 0)
	for rows.Next() {
		var d model.BitrixDepartment
		if err := rows.Scan(&d.BitrixID, &d.ParentBitrixID, &d.Name, &d.Sort); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *BitrixRepository) ListUsers(ctx context.Context, q string, limit int) ([]model.BitrixUser, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := r.pool.Query(ctx, `
		SELECT bitrix_id, email, name, last_name, active, department_ids
		FROM bitrix_users
		WHERE ($1 = '' OR email ILIKE $2 OR name ILIKE $2 OR last_name ILIKE $2)
		ORDER BY last_name, name
		LIMIT $3`, q, "%"+q+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.BitrixUser, 0)
	for rows.Next() {
		var u model.BitrixUser
		if err := rows.Scan(&u.BitrixID, &u.Email, &u.Name, &u.LastName, &u.Active, &u.DepartmentIDs); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *BitrixRepository) InsertRun(ctx context.Context, actorID, status, publicErr string, deps, users int) error {
	var actor any
	if actorID != "" {
		actor = actorID
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO bitrix_sync_runs (actor_user_id, status, error_public, departments_count, users_count, finished_at)
		VALUES ($1, $2, $3, $4, $5, NOW())`, actor, status, publicErr, deps, users)
	return err
}

func (r *BitrixRepository) LastRun(ctx context.Context) (status, publicErr string, finished *time.Time, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT status, error_public, finished_at
		FROM bitrix_sync_runs
		ORDER BY started_at DESC
		LIMIT 1`).Scan(&status, &publicErr, &finished)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", nil, ErrNotFound
		}
		return "", "", nil, err
	}
	return status, publicErr, finished, nil
}
