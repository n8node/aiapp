package config

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
)

const Version = "0.1.0"

type Config struct {
	Port         string   `env:"SERVER_PORT" envDefault:"8080"`
	DatabaseURL  string   `env:"DATABASE_URL,required"`
	Environment  string   `env:"ENVIRONMENT" envDefault:"development"`
	PublicAppURL string   `env:"PUBLIC_APP_URL" envDefault:"http://localhost/app"`
	CORSOrigins  []string `env:"CORS_ORIGINS" envSeparator:"," envDefault:"http://localhost"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}
	for i, origin := range cfg.CORSOrigins {
		cfg.CORSOrigins[i] = strings.TrimSpace(origin)
	}
	return cfg, nil
}

func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.Environment, "production")
}
