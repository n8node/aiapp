package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

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

func (r *InviteRepository) Insert(ctx context.Context, hash, prefix, encrypted, createdBy string) (*model.Invite, error) {
	var inv model.Invite
	err := r.pool.QueryRow(ctx, `
		INSERT INTO invites (code_hash, code_prefix, code_encrypted, created_by_user_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, code_prefix, status, created_by_user_id, used_by_user_id, used_at, expires_at, created_at`,
		hash, prefix, encrypted, createdBy).Scan(
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

type InviteListFilter struct {
	Status      string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UsedFrom    *time.Time
	UsedTo      *time.Time
	UsedEmail   string
	Limit       int
	Offset      int
}

func (r *InviteRepository) List(ctx context.Context, f InviteListFilter) ([]model.Invite, int, error) {
	if f.Limit <= 0 {
		f.Limit = 100
	}
	if f.Limit > 500 {
		f.Limit = 500
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	where := []string{"1=1"}
	args := []any{}
	n := 1
	add := func(clause string, v any) {
		where = append(where, fmt.Sprintf(clause, n))
		args = append(args, v)
		n++
	}
	if f.Status != "" {
		add("i.status = $%d", f.Status)
	}
	if f.CreatedFrom != nil {
		add("i.created_at >= $%d", *f.CreatedFrom)
	}
	if f.CreatedTo != nil {
		add("i.created_at < $%d", *f.CreatedTo)
	}
	if f.UsedFrom != nil {
		add("i.used_at IS NOT NULL AND i.used_at >= $%d", *f.UsedFrom)
	}
	if f.UsedTo != nil {
		add("i.used_at IS NOT NULL AND i.used_at < $%d", *f.UsedTo)
	}
	if email := strings.TrimSpace(f.UsedEmail); email != "" {
		add("u.email ILIKE $%d", "%"+email+"%")
	}
	clause := strings.Join(where, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM invites i
		LEFT JOIN users u ON u.id = i.used_by_user_id
		WHERE `+clause, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitI, offsetI := n, n+1
	args = append(args, f.Limit, f.Offset)
	rows, err := r.pool.Query(ctx, `
		SELECT i.id, i.code_prefix, i.code_encrypted, i.status, i.created_by_user_id,
		       i.used_by_user_id, i.used_at, i.expires_at, i.created_at, u.email
		FROM invites i
		LEFT JOIN users u ON u.id = i.used_by_user_id
		WHERE `+clause+`
		ORDER BY i.created_at DESC
		LIMIT $`+fmt.Sprint(limitI)+` OFFSET $`+fmt.Sprint(offsetI), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []model.Invite
	for rows.Next() {
		var inv model.Invite
		var usedEmail *string
		if err := rows.Scan(
			&inv.ID, &inv.CodePrefix, &inv.CodeEncrypted, &inv.Status, &inv.CreatedBy,
			&inv.UsedBy, &inv.UsedAt, &inv.ExpiresAt, &inv.CreatedAt, &usedEmail,
		); err != nil {
			return nil, 0, err
		}
		inv.UsedByEmail = usedEmail
		out = append(out, inv)
	}
	return out, total, rows.Err()
}

func (r *InviteRepository) DeleteIDs(ctx context.Context, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM invites WHERE id = ANY($1::uuid[])`, ids)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
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

func (r *SessionRepository) ActiveWorkspace(ctx context.Context, sessionID string) (string, error) {
	var id *string
	err := r.pool.QueryRow(ctx, `
		SELECT active_workspace_id FROM sessions WHERE id = $1 AND revoked_at IS NULL`, sessionID).Scan(&id)
	if err != nil {
		return "", err
	}
	if id == nil {
		return "", nil
	}
	return *id, nil
}

func (r *SessionRepository) SetActiveWorkspace(ctx context.Context, sessionID, workspaceID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE sessions SET active_workspace_id = $2, last_active_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL`, sessionID, workspaceID)
	return err
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
