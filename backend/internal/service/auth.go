package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/n8node/aiapp/internal/authn"
	"github.com/n8node/aiapp/internal/cryptoutil"
	"github.com/n8node/aiapp/internal/emaildomain"
	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/password"
	"github.com/n8node/aiapp/internal/repository"
	"github.com/pquerna/otp/totp"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email already registered")
	ErrUserBlocked        = errors.New("account blocked")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInviteRequired     = errors.New("invite required")
	ErrInviteInvalid      = errors.New("invite invalid")
	ErrDomainNotAllowed   = errors.New("email domain not allowed")
	ErrEmailReserved      = errors.New("email reserved")
	ErrTotpRequired       = errors.New("totp required")
	ErrTotpSetupRequired  = errors.New("totp setup required")
	ErrInvalidTotp        = errors.New("invalid totp code")
	ErrTotpAlreadyEnabled = errors.New("totp already enabled")
	ErrForbidden          = errors.New("forbidden")
)

const settingsDomainsKey = "auth.allowed_email_domains"
const tokenTTL = 7 * 24 * time.Hour

var slugSanitizer = regexp.MustCompile(`[^a-z0-9-]+`)
var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type AuthService struct {
	users      *repository.UserRepository
	workspaces *repository.WorkspaceRepository
	invites    *repository.InviteRepository
	settings   *repository.SettingsRepository
	sessions   *repository.SessionRepository
	audit      *repository.AuditRepository
	tokens     *authn.JWT
	totpKey    string
	issuer     string
}

func NewAuthService(
	users *repository.UserRepository,
	workspaces *repository.WorkspaceRepository,
	invites *repository.InviteRepository,
	settings *repository.SettingsRepository,
	sessions *repository.SessionRepository,
	audit *repository.AuditRepository,
	tokens *authn.JWT,
	totpKey, issuer string,
) *AuthService {
	return &AuthService{
		users: users, workspaces: workspaces, invites: invites,
		settings: settings, sessions: sessions, audit: audit,
		tokens: tokens, totpKey: totpKey, issuer: issuer,
	}
}

type AuthResult struct {
	Token      string
	User       model.User
	Workspace  *model.Workspace
	Workspaces []model.Workspace
}

func (s *AuthService) AllowedDomains(ctx context.Context) ([]string, error) {
	raw, err := s.settings.Get(ctx, settingsDomainsKey)
	if err != nil {
		return nil, err
	}
	return emaildomain.ParseList(raw), nil
}

func (s *AuthService) SetAllowedDomains(ctx context.Context, actorID string, domains []string) ([]string, error) {
	encoded := emaildomain.EncodeList(domains)
	parsed := emaildomain.ParseList(encoded)
	if len(parsed) == 0 {
		return nil, ErrInvalidInput
	}
	if err := s.settings.Set(ctx, settingsDomainsKey, encoded); err != nil {
		return nil, err
	}
	s.audit.Write(ctx, actorID, "auth.domains.update", encoded)
	return parsed, nil
}

func (s *AuthService) assertEmailAllowed(ctx context.Context, email string, registering bool) error {
	email = emaildomain.Normalize(email)
	if _, err := mail.ParseAddress(email); err != nil {
		return ErrInvalidInput
	}
	if registering && emaildomain.IsReservedSuperadmin(email) {
		return ErrEmailReserved
	}
	domains, err := s.AllowedDomains(ctx)
	if err != nil {
		return err
	}
	if !emaildomain.Allowed(email, domains) {
		return ErrDomainNotAllowed
	}
	return nil
}

func (s *AuthService) Register(ctx context.Context, email, pass, name, inviteCode string) (*AuthResult, error) {
	email = emaildomain.Normalize(email)
	if err := s.assertEmailAllowed(ctx, email, true); err != nil {
		return nil, err
	}
	if err := password.Validate(pass); err != nil {
		return nil, err
	}
	code := strings.TrimSpace(inviteCode)
	if code == "" {
		return nil, ErrInviteRequired
	}
	inv, err := s.invites.GetActiveByHash(ctx, repository.HashInviteCode(code))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInviteInvalid
		}
		return nil, err
	}

	taken, err := s.users.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrEmailTaken
	}

	hash, err := password.Hash(pass)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		name = strings.Split(email, "@")[0]
	}

	inviteID := inv.ID
	user, err := s.users.Create(ctx, email, hash, strings.TrimSpace(name), &inviteID)
	if err != nil {
		return nil, err
	}
	if err := s.invites.Consume(ctx, inv.ID, user.ID); err != nil {
		return nil, err
	}

	slug, err := s.uniqueSlug(ctx, slugFromEmail(email))
	if err != nil {
		return nil, err
	}
	ws, err := s.workspaces.CreateWithOwner(ctx, "Рабочее пространство", slug, user.ID)
	if err != nil {
		return nil, err
	}
	s.audit.Write(ctx, user.ID, "auth.register", user.Email)
	return s.issue(ctx, &user.User, ws)
}

func (s *AuthService) Login(ctx context.Context, email, pass, totpCode string) (*AuthResult, error) {
	email = emaildomain.Normalize(email)
	if email == "" || pass == "" {
		return nil, ErrInvalidCredentials
	}
	if err := s.assertEmailAllowed(ctx, email, false); err != nil {
		if errors.Is(err, ErrDomainNotAllowed) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	user, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if user.IsBlocked {
		return nil, ErrUserBlocked
	}
	if err := password.Compare(user.PasswordHash, pass); err != nil {
		return nil, ErrInvalidCredentials
	}
	if user.TotpEnabledAt != nil {
		if strings.TrimSpace(totpCode) == "" {
			return nil, ErrTotpRequired
		}
		if err := s.verifyTotp(user, totpCode); err != nil {
			return nil, err
		}
	}
	list, err := s.workspaces.ListForUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	var ws *model.Workspace
	if len(list) > 0 {
		ws = &list[0]
	}
	return s.issue(ctx, &user.User, ws)
}

func (s *AuthService) Me(ctx context.Context, userID string) (*model.User, []model.Workspace, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	if user.IsBlocked {
		return nil, nil, ErrUserBlocked
	}
	list, err := s.workspaces.ListForUser(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	return &user.User, list, nil
}

func (s *AuthService) RequireUser(ctx context.Context, userID string) (*model.UserRecord, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.IsBlocked {
		return nil, ErrUserBlocked
	}
	return user, nil
}

func (s *AuthService) RequireSession(ctx context.Context, userID, sessionID string) (*model.UserRecord, error) {
	user, err := s.RequireUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	ok, err := s.sessions.IsActive(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInvalidCredentials
	}
	_ = s.sessions.Touch(ctx, sessionID)
	return user, nil
}

func (s *AuthService) Logout(ctx context.Context, sessionID string) {
	if sessionID == "" {
		return
	}
	_ = s.sessions.Revoke(ctx, sessionID)
}

func (s *AuthService) SetupTotp(ctx context.Context, userID string) (secret, otpauth string, err error) {
	user, err := s.RequireUser(ctx, userID)
	if err != nil {
		return "", "", err
	}
	if user.TotpEnabledAt != nil {
		return "", "", ErrTotpAlreadyEnabled
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.issuer,
		AccountName: user.Email,
	})
	if err != nil {
		return "", "", err
	}
	enc, err := cryptoutil.Encrypt(key.Secret(), s.totpKey)
	if err != nil {
		return "", "", err
	}
	if err := s.users.SetTotpSecret(ctx, user.ID, enc); err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

func (s *AuthService) ConfirmTotp(ctx context.Context, userID, code string) error {
	user, err := s.RequireUser(ctx, userID)
	if err != nil {
		return err
	}
	if user.TotpEnabledAt != nil {
		return ErrTotpAlreadyEnabled
	}
	if user.TotpSecretEncrypted == "" {
		return ErrTotpSetupRequired
	}
	if err := s.verifyTotp(user, code); err != nil {
		return err
	}
	return s.users.EnableTotp(ctx, user.ID)
}

func (s *AuthService) verifyTotp(user *model.UserRecord, code string) error {
	secret, err := cryptoutil.Decrypt(user.TotpSecretEncrypted, s.totpKey)
	if err != nil {
		return ErrInvalidTotp
	}
	if !totp.Validate(strings.TrimSpace(strings.ReplaceAll(code, " ", "")), secret) {
		return ErrInvalidTotp
	}
	return nil
}

func (s *AuthService) VerifyInvite(ctx context.Context, code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return ErrInviteInvalid
	}
	_, err := s.invites.GetActiveByHash(ctx, repository.HashInviteCode(code))
	if errors.Is(err, repository.ErrNotFound) {
		return ErrInviteInvalid
	}
	return err
}

func (s *AuthService) IssueInvites(ctx context.Context, actorID string, count int) ([]model.IssuedInvite, error) {
	if count < 1 {
		count = 1
	}
	if count > 200 {
		count = 200
	}
	out := make([]model.IssuedInvite, 0, count)
	for i := 0; i < count; i++ {
		raw := make([]byte, 16)
		if _, err := rand.Read(raw); err != nil {
			return nil, err
		}
		code := strings.ToUpper(hex.EncodeToString(raw))
		prefix := code[:4]
		enc, err := cryptoutil.Encrypt(code, s.totpKey)
		if err != nil {
			return nil, err
		}
		inv, err := s.invites.Insert(ctx, repository.HashInviteCode(code), prefix, enc, actorID)
		if err != nil {
			return nil, err
		}
		out = append(out, model.IssuedInvite{Invite: *inv, Code: code})
	}
	s.audit.Write(ctx, actorID, "auth.invite.issue", fmt.Sprintf("count=%d", count))
	return out, nil
}

func (s *AuthService) ListInvites(ctx context.Context, f repository.InviteListFilter) ([]model.Invite, int, error) {
	invites, total, err := s.invites.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	for i := range invites {
		if invites[i].CodeEncrypted == "" {
			continue
		}
		code, err := cryptoutil.Decrypt(invites[i].CodeEncrypted, s.totpKey)
		if err != nil {
			continue
		}
		invites[i].Code = code
		invites[i].CodeEncrypted = ""
	}
	return invites, total, nil
}

func (s *AuthService) DeleteInvites(ctx context.Context, actorID string, ids []string) (int64, error) {
	clean := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || !uuidPattern.MatchString(id) {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		clean = append(clean, id)
	}
	if len(clean) == 0 {
		return 0, ErrInvalidInput
	}
	if len(clean) > 500 {
		clean = clean[:500]
	}
	n, err := s.invites.DeleteIDs(ctx, clean)
	if err != nil {
		return 0, err
	}
	s.audit.Write(ctx, actorID, "auth.invite.delete", fmt.Sprintf("count=%d", n))
	return n, nil
}

func (s *AuthService) RevokeInvite(ctx context.Context, actorID, id string) error {
	if err := s.invites.Revoke(ctx, id); err != nil {
		return err
	}
	s.audit.Write(ctx, actorID, "auth.invite.revoke", id)
	return nil
}

func (s *AuthService) ListUsers(ctx context.Context, q string, blocked, admin *bool, limit, offset int) ([]model.User, int, error) {
	return s.users.List(ctx, q, blocked, admin, limit, offset)
}

func (s *AuthService) SetUserBlocked(ctx context.Context, actorID, userID string, blocked bool) error {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.IsPlatformAdmin {
		return ErrForbidden
	}
	if err := s.users.SetBlocked(ctx, userID, blocked); err != nil {
		return err
	}
	if blocked {
		_ = s.sessions.RevokeAllForUser(ctx, userID)
	}
	action := "auth.user.unblock"
	if blocked {
		action = "auth.user.block"
	}
	s.audit.Write(ctx, actorID, action, userID)
	return nil
}

func (s *AuthService) EnsureSuperAdmin(ctx context.Context, email, pass, name string) (*model.User, bool, error) {
	email = emaildomain.Normalize(email)
	if !emaildomain.IsReservedSuperadmin(email) {
		return nil, false, ErrEmailReserved
	}
	existing, err := s.users.GetByEmail(ctx, email)
	if err == nil {
		if !existing.IsPlatformAdmin {
			if err := s.users.PromotePlatformAdmin(ctx, existing.ID); err != nil {
				return nil, false, err
			}
			existing.IsPlatformAdmin = true
		}
		return &existing.User, false, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, false, err
	}
	if err := password.ValidateSeed(pass); err != nil {
		return nil, false, err
	}
	hash, err := password.Hash(pass)
	if err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(name) == "" {
		name = "Администратор"
	}
	user, err := s.users.CreateSuperadmin(ctx, email, hash, strings.TrimSpace(name))
	if err != nil {
		return nil, false, err
	}
	slug, err := s.uniqueSlug(ctx, "admin")
	if err != nil {
		return nil, false, err
	}
	if _, err := s.workspaces.CreateWithOwner(ctx, "Администрирование", slug, user.ID); err != nil {
		return nil, false, err
	}
	return &user.User, true, nil
}

func (s *AuthService) issue(ctx context.Context, user *model.User, ws *model.Workspace) (*AuthResult, error) {
	sid, err := s.sessions.Create(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	token, err := s.tokens.Issue(user.ID, sid, tokenTTL)
	if err != nil {
		return nil, err
	}
	list := []model.Workspace{}
	if ws != nil {
		list = []model.Workspace{*ws}
	}
	return &AuthResult{Token: token, User: *user, Workspace: ws, Workspaces: list}, nil
}

func slugFromEmail(email string) string {
	local, _, _ := strings.Cut(email, "@")
	s := slugSanitizer.ReplaceAllString(strings.ToLower(local), "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "user"
	}
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}

func (s *AuthService) uniqueSlug(ctx context.Context, base string) (string, error) {
	for i := 0; i < 20; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", base, i+1)
		}
		exists, err := s.workspaces.SlugExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano()), nil
}
