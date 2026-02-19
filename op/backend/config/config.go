package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port             string
	DSN              string
	BaseURL          string
	KeyEncryptionKey string
	FrontendBaseURL  string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:             os.Getenv("OP_BACKEND_PORT"),
		DSN:              os.Getenv("OP_BACKEND_DSN"),
		BaseURL:          os.Getenv("OP_BACKEND_BASE_URL"),
		KeyEncryptionKey: os.Getenv("OP_KEY_ENCRYPTION_KEY"),
		FrontendBaseURL:  os.Getenv("OP_FRONTEND_BASE_URL"),
	}

	if cfg.Port == "" {
		return nil, fmt.Errorf("OP_BACKEND_PORT is required")
	}
	if cfg.DSN == "" {
		return nil, fmt.Errorf("OP_BACKEND_DSN is required")
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("OP_BACKEND_BASE_URL is required")
	}
	if cfg.KeyEncryptionKey == "" {
		return nil, fmt.Errorf("OP_KEY_ENCRYPTION_KEY is required")
	}
	if cfg.FrontendBaseURL == "" {
		return nil, fmt.Errorf("OP_FRONTEND_BASE_URL is required")
	}

	// issuer URL の末尾スラッシュを除去 (OIDC Discovery 1.0 Section 4.1)
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")

	return cfg, nil
}

func (c *Config) IsSecure() bool {
	return strings.HasPrefix(c.BaseURL, "https://")
}
