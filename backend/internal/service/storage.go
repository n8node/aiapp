package service

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/n8node/aiapp/internal/config"
	"github.com/n8node/aiapp/internal/cryptoutil"
	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/repository"
)

var ErrInvalidStorageSettings = errors.New("invalid storage settings")

type StorageSettingsService struct {
	repo   *repository.StorageSettingsRepository
	audit  *repository.AuditRepository
	cfg    *config.Config
	encKey string
}

func NewStorageSettingsService(
	repo *repository.StorageSettingsRepository,
	audit *repository.AuditRepository,
	cfg *config.Config,
	encKey string,
) *StorageSettingsService {
	return &StorageSettingsService{repo: repo, audit: audit, cfg: cfg, encKey: encKey}
}

func (s *StorageSettingsService) GetStored(ctx context.Context) (*model.StorageSettingsRecord, error) {
	rec, err := s.repo.Get(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			def := model.DefaultStorageSettings()
			return &model.StorageSettingsRecord{Config: def}, nil
		}
		return nil, err
	}
	rec.Config.SecretKey = s.decryptSecret(rec.Config.SecretKey)
	return rec, nil
}

func (s *StorageSettingsService) GetEffective(ctx context.Context) (model.StorageSettings, error) {
	rec, err := s.GetStored(ctx)
	if err != nil {
		return model.StorageSettings{}, err
	}
	return rec.Config, nil
}

func (s *StorageSettingsService) GetAdminView(ctx context.Context) (*model.StorageAdminView, error) {
	rec, err := s.GetStored(ctx)
	if err != nil {
		return nil, err
	}
	return s.buildAdminView(rec), nil
}

func (s *StorageSettingsService) Update(ctx context.Context, actorID string, req model.StorageAdminUpdateRequest) (*model.StorageAdminView, error) {
	if err := validateStorageSettings(req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Endpoint) != "" {
		if _, err := parseStorageEndpoint(req.Endpoint, req.UseSSL); err != nil {
			return nil, fmt.Errorf("%w: некорректный endpoint", ErrInvalidStorageSettings)
		}
	}

	rec, err := s.GetStored(ctx)
	if err != nil {
		return nil, err
	}

	cfg := rec.Config
	cfg.Endpoint = strings.TrimSpace(req.Endpoint)
	cfg.Bucket = strings.TrimSpace(req.Bucket)
	cfg.Region = strings.TrimSpace(req.Region)
	cfg.AccessKey = strings.TrimSpace(req.AccessKey)
	cfg.UseSSL = req.UseSSL
	cfg.PathStyle = req.PathStyle
	cfg.Enabled = req.Enabled
	if strings.TrimSpace(req.SecretKey) != "" {
		cfg.SecretKey = strings.TrimSpace(req.SecretKey)
	}
	if cfg.Region == "" {
		cfg.Region = "ru-central1"
	}
	if StorageConfigured(cfg) {
		cfg.Enabled = true
	}

	persisted := cfg
	enc, err := s.encryptSecret(cfg.SecretKey)
	if err != nil {
		return nil, err
	}
	persisted.SecretKey = enc
	updated, err := s.repo.Update(ctx, persisted)
	if err != nil {
		return nil, err
	}
	updated.Config.SecretKey = cfg.SecretKey
	host := storageHost(cfg.Endpoint)
	s.audit.Write(ctx, actorID, "storage.settings.save", host+"/"+cfg.Bucket)
	return s.buildAdminView(updated), nil
}

func (s *StorageSettingsService) TestConnection(ctx context.Context) (*model.StorageTestResult, error) {
	rec, err := s.GetStored(ctx)
	if err != nil {
		return nil, err
	}
	st := rec.Config
	if strings.TrimSpace(st.Endpoint) == "" {
		return &model.StorageTestResult{OK: false, Message: "Укажите endpoint (URL)"}, nil
	}
	if strings.TrimSpace(st.Bucket) == "" {
		return &model.StorageTestResult{OK: false, Message: "Укажите имя бакета"}, nil
	}
	if strings.TrimSpace(st.AccessKey) == "" {
		return &model.StorageTestResult{OK: false, Message: "Укажите Access Key ID"}, nil
	}
	if strings.TrimSpace(st.SecretKey) == "" {
		return &model.StorageTestResult{OK: false, Message: "Укажите Secret Access Key"}, nil
	}

	client, err := newS3Client(st)
	if err != nil {
		return &model.StorageTestResult{OK: false, Message: "Некорректный endpoint"}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(st.Bucket)})
	if err != nil {
		return &model.StorageTestResult{
			OK:      false,
			Message: "Не удалось подключиться к бакету: " + sanitizeS3Error(err),
		}, nil
	}

	if StorageConfigured(st) && !st.Enabled {
		st.Enabled = true
		enc, encErr := s.encryptSecret(st.SecretKey)
		if encErr == nil {
			saved := st
			saved.SecretKey = enc
			_, _ = s.repo.Update(ctx, saved)
		}
	}

	return &model.StorageTestResult{
		OK:      true,
		Message: fmt.Sprintf("Соединение успешно. Бакет «%s» доступен.", st.Bucket),
	}, nil
}

func (s *StorageSettingsService) buildAdminView(rec *model.StorageSettingsRecord) *model.StorageAdminView {
	st := rec.Config
	origins := buildCORSOrigins(s.cfg.CORSOrigins, s.cfg.PublicAppURL)
	return &model.StorageAdminView{
		Endpoint:      st.Endpoint,
		Bucket:        st.Bucket,
		Region:        st.Region,
		AccessKey:     st.AccessKey,
		SecretKeySet:  strings.TrimSpace(st.SecretKey) != "",
		SecretKeyHint: maskSecret(st.SecretKey),
		UseSSL:        st.UseSSL,
		PathStyle:     st.PathStyle,
		Enabled:       st.Enabled,
		CORSOrigins:   origins,
		CORSXML:       buildCORSXML(origins),
		UpdatedAt:     rec.UpdatedAt,
	}
}

func (s *StorageSettingsService) encryptSecret(plain string) (string, error) {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return "", nil
	}
	return cryptoutil.Encrypt(plain, s.encKey)
}

func (s *StorageSettingsService) decryptSecret(stored string) string {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return ""
	}
	plain, err := cryptoutil.Decrypt(stored, s.encKey)
	if err != nil {
		return ""
	}
	return plain
}

func validateStorageSettings(req model.StorageAdminUpdateRequest) error {
	if !req.Enabled {
		return nil
	}
	if strings.TrimSpace(req.Endpoint) == "" || strings.TrimSpace(req.Bucket) == "" || strings.TrimSpace(req.AccessKey) == "" {
		return fmt.Errorf("%w: заполните endpoint, бакет и ключ доступа", ErrInvalidStorageSettings)
	}
	return nil
}

func buildCORSOrigins(allowlist []string, publicAppURL string) []string {
	seen := map[string]struct{}{}
	var origins []string
	add := func(o string) {
		o = strings.TrimSpace(o)
		if o == "" {
			return
		}
		if _, ok := seen[o]; ok {
			return
		}
		seen[o] = struct{}{}
		origins = append(origins, o)
	}
	for _, o := range allowlist {
		add(o)
	}
	if u, err := url.Parse(publicAppURL); err == nil && u.Scheme != "" && u.Host != "" {
		add(u.Scheme + "://" + u.Host)
	}
	if len(origins) == 0 {
		add("http://localhost")
	}
	return origins
}

func buildCORSXML(origins []string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<CORSConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">` + "\n")
	b.WriteString("  <CORSRule>\n")
	for _, o := range origins {
		b.WriteString("    <AllowedOrigin>" + html.EscapeString(o) + "</AllowedOrigin>\n")
	}
	for _, method := range []string{"GET", "PUT", "POST", "DELETE", "HEAD"} {
		b.WriteString("    <AllowedMethod>" + method + "</AllowedMethod>\n")
	}
	b.WriteString("    <AllowedHeader>*</AllowedHeader>\n")
	b.WriteString("    <ExposeHeader>ETag</ExposeHeader>\n")
	b.WriteString("  </CORSRule>\n")
	b.WriteString("</CORSConfiguration>")
	return b.String()
}

func newS3Client(st model.StorageSettings) (*s3.Client, error) {
	endpoint, err := parseStorageEndpoint(st.Endpoint, st.UseSSL)
	if err != nil {
		return nil, err
	}
	region := strings.TrimSpace(st.Region)
	if region == "" {
		region = "us-east-1"
	}
	awsCfg := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(st.AccessKey, st.SecretKey, ""),
	}
	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = st.PathStyle
	}), nil
}

func parseStorageEndpoint(raw string, useSSL bool) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("empty endpoint")
	}
	if !strings.Contains(raw, "://") {
		scheme := "http"
		if useSSL {
			scheme = "https"
		}
		raw = scheme + "://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", errors.New("parse endpoint")
	}
	if u.User != nil || u.Fragment != "" {
		return "", errors.New("invalid endpoint")
	}
	if !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		return "", errors.New("scheme")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return "", errors.New("host")
	}
	if ip := net.ParseIP(host); ip != nil && (ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()) {
		return "", errors.New("blocked host")
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	return u.Scheme + "://" + u.Host + u.Path, nil
}

func storageHost(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Hostname()
}

func sanitizeS3Error(err error) string {
	msg := err.Error()
	lower := strings.ToLower(msg)
	if i := strings.Index(lower, "secret"); i >= 0 {
		return "ошибка хранилища"
	}
	if i := strings.Index(lower, "credential"); i >= 0 {
		return "хранилище отклонило ключи доступа"
	}
	if len(msg) > 180 {
		return msg[:180] + "…"
	}
	return msg
}

func maskSecret(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return ""
	}
	if len(v) <= 4 {
		return "••••"
	}
	return v[:2] + "••••" + v[len(v)-2:]
}

func StorageConfigured(st model.StorageSettings) bool {
	return strings.TrimSpace(st.Endpoint) != "" &&
		strings.TrimSpace(st.Bucket) != "" &&
		strings.TrimSpace(st.AccessKey) != "" &&
		strings.TrimSpace(st.SecretKey) != ""
}

func IsStorageEnabled(st model.StorageSettings) bool {
	return st.Enabled && StorageConfigured(st)
}
