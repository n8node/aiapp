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

var ErrNotFound = errors.New("not found")

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

const userCols = `id, email, password_hash, name, locale, timezone, is_platform_admin, is_blocked,
email_verified_at, totp_secret_encrypted, totp_enabled_at, registered_via_invite_id, created_at, updated_at`

func scanUser(row pgx.Row) (*model.UserRecord, error) {
	var u model.UserRecord
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Locale, &u.Timezone,
		&u.IsPlatformAdmin, &u.IsBlocked, &u.EmailVerifiedAt, &u.TotpSecretEncrypted,
		&u.TotpEnabledAt, &u.RegisteredViaInviteID, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.TotpEnabled = u.TotpEnabledAt != nil
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, email, passwordHash, name string, inviteID *string) (*model.UserRecord, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, name, email_verified_at, registered_via_invite_id)
		VALUES ($1, $2, $3, NOW(), $4)
		RETURNING `+userCols, email, passwordHash, name, inviteID)
	return scanUser(row)
}

func (r *UserRepository) CreateSuperadmin(ctx context.Context, email, passwordHash, name string) (*model.UserRecord, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, name, is_platform_admin, email_verified_at)
		VALUES ($1, $2, $3, true, NOW())
		RETURNING `+userCols, email, passwordHash, name)
	return scanUser(row)
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.UserRecord, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE email = $1`, email)
	return scanUser(row)
}

func (r *UserRepository) IDsByEmails(ctx context.Context, emails []string) (map[string]string, error) {
	out := map[string]string{}
	if len(emails) == 0 {
		return out, nil
	}
	lowered := make([]string, 0, len(emails))
	seen := map[string]struct{}{}
	for _, e := range emails {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		lowered = append(lowered, e)
	}
	if len(lowered) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT id, lower(email) FROM users WHERE lower(email) = ANY($1)`, lowered)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, email string
		if err := rows.Scan(&id, &email); err != nil {
			return nil, err
		}
		out[email] = id
	}
	return out, rows.Err()
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*model.UserRecord, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id = $1`, id)
	return scanUser(row)
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, email).Scan(&ok)
	return ok, err
}

func (r *UserRepository) SetBlocked(ctx context.Context, id string, blocked bool) error {
	tag, err := r.pool.Exec(ctx, `UPDATE users SET is_blocked = $2, updated_at = NOW() WHERE id = $1`, id, blocked)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepository) SetTotpSecret(ctx context.Context, id, enc string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET totp_secret_encrypted = $2, totp_enabled_at = NULL, updated_at = NOW()
		WHERE id = $1`, id, enc)
	return err
}

func (r *UserRepository) EnableTotp(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET totp_enabled_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND totp_secret_encrypted <> ''`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepository) PromotePlatformAdmin(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET is_platform_admin = true, updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *UserRepository) List(ctx context.Context, q string, blocked *bool, admin *bool, limit, offset int) ([]model.User, int, error) {
	where := []string{"1=1"}
	args := []any{}
	n := 1
	if strings.TrimSpace(q) != "" {
		where = append(where, fmt.Sprintf("(email ILIKE $%d OR name ILIKE $%d)", n, n))
		args = append(args, "%"+strings.TrimSpace(q)+"%")
		n++
	}
	if blocked != nil {
		where = append(where, fmt.Sprintf("is_blocked = $%d", n))
		args = append(args, *blocked)
		n++
	}
	if admin != nil {
		where = append(where, fmt.Sprintf("is_platform_admin = $%d", n))
		args = append(args, *admin)
		n++
	}
	clause := strings.Join(where, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE `+clause, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	limitArg := n
	offsetArg := n + 1
	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, `
		SELECT id, email, name, locale, timezone, is_platform_admin, is_blocked,
		       email_verified_at, totp_enabled_at, created_at, updated_at
		FROM users WHERE `+clause+`
		ORDER BY created_at DESC
		LIMIT $`+fmt.Sprint(limitArg)+` OFFSET $`+fmt.Sprint(offsetArg), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []model.User
	for rows.Next() {
		var u model.User
		var totpAt *time.Time
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Locale, &u.Timezone, &u.IsPlatformAdmin,
			&u.IsBlocked, &u.EmailVerifiedAt, &totpAt, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		u.TotpEnabled = totpAt != nil
		out = append(out, u)
	}
	return out, total, rows.Err()
}

type WorkspaceRepository struct {
	pool *pgxpool.Pool
}

func NewWorkspaceRepository(pool *pgxpool.Pool) *WorkspaceRepository {
	return &WorkspaceRepository{pool: pool}
}

func (r *WorkspaceRepository) CreateWithOwner(ctx context.Context, name, slug, ownerID string) (*model.Workspace, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var ws model.Workspace
	err = tx.QueryRow(ctx, `
		INSERT INTO workspaces (name, slug, owner_id) VALUES ($1, $2, $3)
		RETURNING id, name, slug, owner_id, created_at`, name, slug, ownerID).Scan(
		&ws.ID, &ws.Name, &ws.Slug, &ws.OwnerID, &ws.CreatedAt)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'employee')`,
		ws.ID, ownerID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	ws.Role = "employee"
	return &ws, nil
}

func (r *WorkspaceRepository) ListForUser(ctx context.Context, userID string) ([]model.Workspace, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT w.id, w.name, w.slug, w.owner_id, w.created_at, m.role
		FROM workspaces w
		JOIN workspace_members m ON m.workspace_id = w.id
		WHERE m.user_id = $1
		ORDER BY w.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Workspace
	for rows.Next() {
		var w model.Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Slug, &w.OwnerID, &w.CreatedAt, &w.Role); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *WorkspaceRepository) SlugExists(ctx context.Context, slug string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM workspaces WHERE slug = $1)`, slug).Scan(&ok)
	return ok, err
}
