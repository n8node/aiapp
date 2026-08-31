package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/n8node/aiapp/internal/bitrix"
	"github.com/n8node/aiapp/internal/bitrixurl"
	"github.com/n8node/aiapp/internal/cryptoutil"
	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/repository"
)

const (
	bitrixWebhookKey = "bitrix.webhook_encrypted"
	bitrixHostKey    = "bitrix.portal_host"
)

var (
	ErrBitrixNotConfigured = errors.New("bitrix not configured")
	ErrBitrixInvalidURL    = errors.New("bitrix invalid url")
	ErrBitrixBlockedHost   = errors.New("bitrix blocked host")
	ErrBitrixRequest       = errors.New("bitrix request failed")
	ErrBitrixBusy          = errors.New("bitrix sync busy")
)

type BitrixService struct {
	settings *repository.SettingsRepository
	store    *repository.BitrixRepository
	audit    *repository.AuditRepository
	client   *bitrix.Client
	encKey   string
	mu       sync.Mutex
}

func NewBitrixService(
	settings *repository.SettingsRepository,
	store *repository.BitrixRepository,
	audit *repository.AuditRepository,
	encKey string,
) *BitrixService {
	return &BitrixService{
		settings: settings,
		store:    store,
		audit:    audit,
		client:   bitrix.NewClient(),
		encKey:   encKey,
	}
}

func (s *BitrixService) Status(ctx context.Context) (*model.BitrixStatus, error) {
	st := &model.BitrixStatus{}
	host, err := s.settings.Get(ctx, bitrixHostKey)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	enc, err := s.settings.Get(ctx, bitrixWebhookKey)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	st.Configured = enc != ""
	st.PortalHost = host
	if enc != "" {
		if raw, decErr := cryptoutil.Decrypt(enc, s.encKey); decErr == nil {
			st.WebhookMasked = bitrixurl.MaskWebhook(raw)
		} else {
			st.WebhookMasked = "••••"
		}
	}
	st.DepartmentsCount, st.UsersCount, err = s.store.Counts(ctx)
	if err != nil {
		return nil, err
	}
	status, pubErr, finished, err := s.store.LastRun(ctx)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	if err == nil {
		st.LastSyncStatus = status
		st.LastSyncError = pubErr
		st.LastSyncAt = finished
	}
	return st, nil
}

func (s *BitrixService) SaveWebhook(ctx context.Context, actorID, raw string) (*model.BitrixStatus, error) {
	u, err := bitrixurl.NormalizeWebhook(raw)
	if err != nil {
		if strings.Contains(err.Error(), "blocked") {
			return nil, ErrBitrixBlockedHost
		}
		return nil, ErrBitrixInvalidURL
	}
	if err := bitrixurl.AssertPublicResolved(u.Hostname()); err != nil {
		if strings.Contains(err.Error(), "resolve") {
			return nil, ErrBitrixInvalidURL
		}
		return nil, ErrBitrixBlockedHost
	}
	enc, err := cryptoutil.Encrypt(u.String(), s.encKey)
	if err != nil {
		return nil, err
	}
	if err := s.settings.Set(ctx, bitrixWebhookKey, enc); err != nil {
		return nil, err
	}
	if err := s.settings.Set(ctx, bitrixHostKey, u.Hostname()); err != nil {
		return nil, err
	}
	s.audit.Write(ctx, actorID, "bitrix.webhook.save", u.Hostname())
	return s.Status(ctx)
}

func (s *BitrixService) Disconnect(ctx context.Context, actorID string) (*model.BitrixStatus, error) {
	if err := s.settings.Set(ctx, bitrixWebhookKey, ""); err != nil {
		return nil, err
	}
	if err := s.settings.Set(ctx, bitrixHostKey, ""); err != nil {
		return nil, err
	}
	s.audit.Write(ctx, actorID, "bitrix.webhook.disconnect", "")
	return s.Status(ctx)
}

func (s *BitrixService) webhookURL(ctx context.Context) (string, error) {
	enc, err := s.settings.Get(ctx, bitrixWebhookKey)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", ErrBitrixNotConfigured
		}
		return "", err
	}
	if enc == "" {
		return "", ErrBitrixNotConfigured
	}
	raw, err := cryptoutil.Decrypt(enc, s.encKey)
	if err != nil {
		return "", ErrBitrixNotConfigured
	}
	return raw, nil
}

func wrapBitrix(err error) error {
	if err == nil {
		return nil
	}
	return &BitrixAPIError{Public: bitrix.PublicError(err)}
}

type BitrixAPIError struct {
	Public string
}

func (e *BitrixAPIError) Error() string {
	if e != nil && e.Public != "" {
		return e.Public
	}
	return ErrBitrixRequest.Error()
}

func (e *BitrixAPIError) Unwrap() error {
	return ErrBitrixRequest
}

func (s *BitrixService) Test(ctx context.Context) error {
	raw, err := s.webhookURL(ctx)
	if err != nil {
		return err
	}
	u, err := bitrixurl.NormalizeWebhook(raw)
	if err != nil {
		return ErrBitrixInvalidURL
	}
	if err := s.client.CurrentUser(ctx, u); err != nil {
		return wrapBitrix(err)
	}
	if _, err := s.client.ListDepartments(ctx, u); err != nil {
		return wrapBitrix(err)
	}
	return nil
}

func (s *BitrixService) Sync(ctx context.Context, actorID string) (*model.BitrixSyncResult, error) {
	if !s.mu.TryLock() {
		return nil, ErrBitrixBusy
	}
	defer s.mu.Unlock()

	raw, err := s.webhookURL(ctx)
	if err != nil {
		return nil, err
	}
	u, err := bitrixurl.NormalizeWebhook(raw)
	if err != nil {
		return nil, ErrBitrixInvalidURL
	}

	parent := ctx
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	deps, depErr := s.client.ListDepartments(ctx, u)
	users, userErr := s.client.ListUsers(ctx, u)
	if userErr != nil {
		msg := "Не удалось получить сотрудников"
		if depErr != nil {
			msg = bitrix.PublicError(depErr)
		} else {
			msg = bitrix.PublicError(userErr)
		}
		_ = s.store.InsertRun(parent, actorID, "FAILED", msg, 0, 0)
		if depErr != nil {
			return nil, wrapBitrix(depErr)
		}
		return nil, wrapBitrix(userErr)
	}
	warning := ""
	if depErr != nil {
		warning = bitrix.PublicError(depErr)
		deps = departmentsFromUsers(users)
	}
	if err := s.store.ReplaceSnapshot(ctx, deps, users); err != nil {
		_ = s.store.InsertRun(parent, actorID, "FAILED", "Не удалось сохранить структуру", 0, 0)
		return nil, err
	}
	status := "SUCCESS"
	if warning != "" {
		status = "PARTIAL"
	}
	_ = s.store.InsertRun(parent, actorID, status, warning, len(deps), len(users))
	s.audit.Write(ctx, actorID, "bitrix.sync", u.Hostname())
	return &model.BitrixSyncResult{Departments: len(deps), Users: len(users)}, nil
}

func departmentsFromUsers(users []bitrix.User) []bitrix.Department {
	seen := map[int64]struct{}{}
	var out []bitrix.Department
	for _, u := range users {
		for _, id := range u.DepartmentIDs {
			if id <= 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, bitrix.Department{
				ID:   id,
				Name: "Отдел " + strconv.FormatInt(id, 10),
				Sort: int(id),
			})
		}
	}
	return out
}

func (s *BitrixService) Departments(ctx context.Context) ([]model.BitrixDepartment, error) {
	return s.store.ListDepartments(ctx)
}

func (s *BitrixService) Users(ctx context.Context, q string) ([]model.BitrixUser, error) {
	return s.store.ListUsers(ctx, strings.TrimSpace(q), 200)
}
