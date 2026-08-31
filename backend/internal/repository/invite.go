package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/n8node/aiapp/internal/model"
)

func HashInviteCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

type InviteRepository struct {
	pool *pgxpool.Pool
}

func NewInviteRepository(pool *pgxpool.Pool) *InviteRepository {
	return &InviteRepository{pool: pool}
}

func (r *InviteRepository) Insert(ctx context.Context, hash, prefix, createdBy string) (*model.Invite, error) {
	var inv model.Invite
	err := r.pool.QueryRow(ctx, `
		INSERT INTO invites (code_hash, code_prefix, created_by_user_id)
		VALUES ($1, $2, $3)
		RETURNING id, code_prefix, status, created_by_user_id, used_by_user_id, used_at, expires_at, created_at`,
		hash, prefix, createdBy).Scan(
		&inv.ID, &inv.CodePrefix, &inv.Status, &inv.CreatedBy, &inv.UsedBy, &inv.UsedAt, &inv.ExpiresAt, &inv.CreatedAt)
	return &inv, err
}

func (r *InviteRepository) GetActiveByHash(ctx context.Context, hash string) (*model.Invite, error) {
	var inv model.Invite
	err := r.pool.QueryRow(ctx, `
		SELECT id, code_prefix, status, created_by_user_id, used_by_user_id, used_at, expires_at, created_at
		FROM invites WHERE code_hash = $1 AND status = 'ACTIVE'`, hash).Scan(
		&inv.ID, &inv.CodePrefix, &inv.Status, &inv.CreatedBy, &inv.UsedBy, &inv.UsedAt, &inv.ExpiresAt, &inv.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &inv, err
}

func (r *InviteRepository) Consume(ctx context.Context, id, userID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE invites
		SET status = 'USED', used_by_user_id = $2, used_at = NOW()
		WHERE id = $1 AND status = 'ACTIVE'`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *InviteRepository) Revoke(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE invites SET status = 'REVOKED' WHERE id = $1 AND status = 'ACTIVE'`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *InviteRepository) List(ctx context.Context, status string, limit, offset int) ([]model.Invite, int, error) {
	where := "1=1"
	args := []any{}
	n := 1
	if status != "" {
		where = fmt.Sprintf("status = $%d", n)
		args = append(args, status)
		n++
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM invites WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	limitI, offsetI := n, n+1
	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, `
		SELECT id, code_prefix, status, created_by_user_id, used_by_user_id, used_at, expires_at, created_at
		FROM invites WHERE `+where+`
		ORDER BY created_at DESC
		LIMIT $`+fmt.Sprint(limitI)+` OFFSET $`+fmt.Sprint(offsetI), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []model.Invite
	for rows.Next() {
		var inv model.Invite
		if err := rows.Scan(&inv.ID, &inv.CodePrefix, &inv.Status, &inv.CreatedBy, &inv.UsedBy, &inv.UsedAt, &inv.ExpiresAt, &inv.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, inv)
	}
	return out, total, rows.Err()
}

type SettingsRepository struct {
	pool *pgxpool.Pool
}

func NewSettingsRepository(pool *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{pool: pool}
}

func (r *SettingsRepository) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := r.pool.QueryRow(ctx, `SELECT value FROM app_settings WHERE key = $1`, key).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return value, err
}

func (r *SettingsRepository) Set(ctx context.Context, key, value string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO app_settings (key, value, updated_at) VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`, key, value)
	return err
}

type SessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

func (r *SessionRepository) Create(ctx context.Context, userID string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO sessions (user_id) VALUES ($1) RETURNING id`, userID).Scan(&id)
	return id, err
}

func (r *SessionRepository) Revoke(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW() WHERE id = $1 AND revoked_at IS NULL`, id)
	return err
}

func (r *SessionRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}

func (r *SessionRepository) Touch(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE sessions SET last_active_at = NOW() WHERE id = $1 AND revoked_at IS NULL`, id)
	return err
}

func (r *SessionRepository) IsActive(ctx context.Context, id, userID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM sessions WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL)`,
		id, userID).Scan(&ok)
	return ok, err
}

type AuditRepository struct {
	pool *pgxpool.Pool
}

func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

func (r *AuditRepository) Write(ctx context.Context, actorID, action, target string) {
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO audit_events (actor_user_id, action, target) VALUES ($1, $2, $3)`, actorID, action, target)
}
