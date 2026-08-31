package config

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
)

const Version = "0.1.0"

type Config struct {
	Port               string   `env:"SERVER_PORT" envDefault:"8080"`
	DatabaseURL        string   `env:"DATABASE_URL,required"`
	Environment        string   `env:"ENVIRONMENT" envDefault:"development"`
	PublicAppURL       string   `env:"PUBLIC_APP_URL" envDefault:"http://localhost/app"`
	CORSOrigins        []string `env:"CORS_ORIGINS" envSeparator:"," envDefault:"http://localhost"`
	JWTSecret          string   `env:"JWT_SECRET" envDefault:"dev-only-change-me"`
	TOTPKey            string   `env:"TOTP_KEY" envDefault:""`
	CookieSecure       bool     `env:"COOKIE_SECURE" envDefault:"false"`
	SuperadminEmail    string   `env:"SUPERADMIN_EMAIL"`
	SuperadminPassword string   `env:"SUPERADMIN_PASSWORD"`
	SuperadminName     string   `env:"SUPERADMIN_NAME" envDefault:"George"`
	ExtractURL         string   `env:"EXTRACT_URL"`
	ExtractToken       string   `env:"EXTRACT_TOKEN"`
	GatewayURL         string   `env:"GATEWAY_URL"`
	GatewayToken       string   `env:"GATEWAY_TOKEN"`
	StudioURL          string   `env:"STUDIO_URL"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}
	for i, origin := range cfg.CORSOrigins {
		cfg.CORSOrigins[i] = strings.TrimSpace(origin)
	}
	if cfg.TOTPKey == "" {
		cfg.TOTPKey = cfg.JWTSecret
	}
	if cfg.ExtractToken == "" {
		cfg.ExtractToken = cfg.JWTSecret
	}
	if cfg.GatewayToken == "" {
		cfg.GatewayToken = cfg.JWTSecret
	}
	cfg.ExtractURL = strings.TrimRight(strings.TrimSpace(cfg.ExtractURL), "/")
	cfg.GatewayURL = strings.TrimRight(strings.TrimSpace(cfg.GatewayURL), "/")
	cfg.StudioURL = strings.TrimRight(strings.TrimSpace(cfg.StudioURL), "/")
	if cfg.IsProduction() {
		cfg.CookieSecure = true
		if cfg.JWTSecret == "" || cfg.JWTSecret == "dev-only-change-me" {
			return nil, fmt.Errorf("JWT_SECRET must be set in production")
		}
	}
	return cfg, nil
}

func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.Environment, "production")
}
