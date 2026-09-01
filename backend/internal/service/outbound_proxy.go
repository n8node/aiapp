package service

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/n8node/aiapp/internal/model"
	"github.com/n8node/aiapp/internal/repository"
)

const outboundProxySettingsKey = "outbound.hf_proxy"

var ErrInvalidOutboundProxy = errors.New("invalid outbound proxy")
var ErrOutboundProxyNotEnabled = errors.New("outbound proxy disabled")
var ErrInternalUnauthorized = errors.New("internal unauthorized")

type OutboundProxyService struct {
	settings *repository.SettingsRepository
	audit    *repository.AuditRepository
	internal string
}

func NewOutboundProxyService(settings *repository.SettingsRepository, audit *repository.AuditRepository, internalToken string) *OutboundProxyService {
	return &OutboundProxyService{settings: settings, audit: audit, internal: strings.TrimSpace(internalToken)}
}

func (s *OutboundProxyService) Get(ctx context.Context) (model.OutboundProxySettings, error) {
	raw, err := s.settings.Get(ctx, outboundProxySettingsKey)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.DefaultOutboundProxySettings(), nil
		}
		return model.OutboundProxySettings{}, err
	}
	cfg := model.DefaultOutboundProxySettings()
	if strings.TrimSpace(raw) == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return model.DefaultOutboundProxySettings(), nil
	}
	cfg.ProxyURLs = normalizeProxyURLs(cfg.ProxyURLs)
	if cfg.ProxyEnabled && cfg.ProxyActiveURL == "" && len(cfg.ProxyURLs) > 0 {
		cfg.ProxyActiveURL = cfg.ProxyURLs[0]
	}
	return cfg, nil
}

func (s *OutboundProxyService) GetAdminView(ctx context.Context) (model.OutboundProxyAdminView, error) {
	cfg, err := s.Get(ctx)
	if err != nil {
		return model.OutboundProxyAdminView{}, err
	}
	return s.toView(cfg), nil
}

func (s *OutboundProxyService) Update(ctx context.Context, actorID string, cfg model.OutboundProxySettings) (model.OutboundProxyAdminView, error) {
	cfg.ProxyURLs = normalizeProxyURLs(cfg.ProxyURLs)
	cfg.ProxyActiveURL = strings.TrimSpace(cfg.ProxyActiveURL)
	if cfg.ProxyEnabled {
		if len(cfg.ProxyURLs) == 0 {
			return model.OutboundProxyAdminView{}, fmt.Errorf("%w: укажите хотя бы один HTTP-прокси", ErrInvalidOutboundProxy)
		}
		for _, raw := range cfg.ProxyURLs {
			if _, err := parseHTTPProxyURL(raw); err != nil {
				return model.OutboundProxyAdminView{}, fmt.Errorf("%w: только http://user:pass@host:port", ErrInvalidOutboundProxy)
			}
		}
		if cfg.ProxyActiveURL == "" || !containsProxyURL(cfg.ProxyURLs, cfg.ProxyActiveURL) {
			cfg.ProxyActiveURL = cfg.ProxyURLs[0]
		}
	} else if cfg.ProxyActiveURL != "" && !containsProxyURL(cfg.ProxyURLs, cfg.ProxyActiveURL) {
		cfg.ProxyActiveURL = ""
	}
	body, err := json.Marshal(cfg)
	if err != nil {
		return model.OutboundProxyAdminView{}, err
	}
	if err := s.settings.Set(ctx, outboundProxySettingsKey, string(body)); err != nil {
		return model.OutboundProxyAdminView{}, err
	}
	masked := "off"
	if cfg.ProxyEnabled {
		masked = maskProxyURLForError(s.activeURL(cfg))
	}
	s.audit.Write(ctx, actorID, "outbound.proxy.save", masked)
	return s.toView(cfg), nil
}

func (s *OutboundProxyService) InternalURL(ctx context.Context, token string) (string, error) {
	if s.internal == "" || subtle.ConstantTimeCompare([]byte(token), []byte(s.internal)) != 1 {
		return "", ErrInternalUnauthorized
	}
	cfg, err := s.Get(ctx)
	if err != nil {
		return "", err
	}
	if !cfg.ProxyEnabled {
		return "", nil
	}
	return s.activeURL(cfg), nil
}

func (s *OutboundProxyService) Test(ctx context.Context) (model.OutboundProxyTestResult, error) {
	cfg, err := s.Get(ctx)
	if err != nil {
		return model.OutboundProxyTestResult{}, err
	}
	if !cfg.ProxyEnabled {
		return model.OutboundProxyTestResult{}, ErrOutboundProxyNotEnabled
	}
	proxyURL := s.activeURL(cfg)
	if proxyURL == "" {
		return model.OutboundProxyTestResult{}, fmt.Errorf("%w: нет активного URL", ErrInvalidOutboundProxy)
	}
	client, err := httpClientForProxy(&http.Client{Timeout: 25 * time.Second}, proxyURL)
	if err != nil {
		return model.OutboundProxyTestResult{}, fmt.Errorf("%w: %v", ErrInvalidOutboundProxy, err)
	}

	hub, err := s.probe(ctx, client, "https://huggingface.co/", true)
	if err != nil {
		return model.OutboundProxyTestResult{
			OK:      false,
			Message: "Прокси не открыл huggingface.co: " + err.Error(),
			Hub:     err.Error(),
		}, nil
	}
	cdn, err := s.probe(ctx, client, "https://huggingface.co/unsloth/Qwen3.6-27B-MTP-GGUF/resolve/main/Qwen3.6-27B-UD-Q4_K_XL.gguf", false)
	if err != nil {
		return model.OutboundProxyTestResult{
			OK:      false,
			Message: "Hub через прокси отвечает, CDN/resolve — нет: " + err.Error(),
			Hub:     hub,
			CDN:     err.Error(),
		}, nil
	}
	return model.OutboundProxyTestResult{
		OK:      true,
		Message: "Прокси открывает Hugging Face. Токен модели берётся из настроек Studio. Перезапустите unsloth-studio, чтобы качание шло через прокси.",
		Hub:     hub,
		CDN:     cdn,
	}, nil
}

func (s *OutboundProxyService) probe(ctx context.Context, client *http.Client, rawURL string, follow bool) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "RigIntel-ProxyProbe/1.0")
	transport := client.Transport
	probeClient := &http.Client{
		Timeout:   25 * time.Second,
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			if follow {
				return nil
			}
			return http.ErrUseLastResponse
		},
	}
	resp, err := probeClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	_, _ = io.CopyN(io.Discard, resp.Body, 2048)
	if (resp.StatusCode >= 200 && resp.StatusCode < 400) || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return resp.Status, nil
	}
	return "", fmt.Errorf("HTTP %s", resp.Status)
}

func (s *OutboundProxyService) activeURL(cfg model.OutboundProxySettings) string {
	ordered := proxyOrder(cfg.ProxyActiveURL, cfg.ProxyURLs)
	if len(ordered) == 0 {
		return ""
	}
	return ordered[0]
}

func (s *OutboundProxyService) toView(cfg model.OutboundProxySettings) model.OutboundProxyAdminView {
	masked := ""
	if cfg.ProxyEnabled {
		masked = maskProxyURLForError(s.activeURL(cfg))
	}
	return model.OutboundProxyAdminView{
		ProxyEnabled:   cfg.ProxyEnabled,
		ProxyActiveURL: cfg.ProxyActiveURL,
		ProxyURLs:      cfg.ProxyURLs,
		ActiveMasked:   masked,
	}
}
