package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port             string
	DSN              string
	IssuerBaseURL    string
	KeyEncryptionKey string
	LoginPageURL     string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:             os.Getenv("OP_BACKEND_PORT"),
		DSN:              os.Getenv("OP_BACKEND_DSN"),
		IssuerBaseURL:    os.Getenv("OP_BACKEND_ISSUER_BASE_URL"),
		KeyEncryptionKey: os.Getenv("OP_KEY_ENCRYPTION_KEY"),
		LoginPageURL:     os.Getenv("OP_LOGIN_PAGE_URL"),
	}

	if cfg.Port == "" {
		return nil, fmt.Errorf("OP_BACKEND_PORT is required")
	}
	if cfg.DSN == "" {
		return nil, fmt.Errorf("OP_BACKEND_DSN is required")
	}
	if cfg.IssuerBaseURL == "" {
		return nil, fmt.Errorf("OP_BACKEND_ISSUER_BASE_URL is required")
	}
	if cfg.KeyEncryptionKey == "" {
		return nil, fmt.Errorf("OP_KEY_ENCRYPTION_KEY is required")
	}
	if cfg.LoginPageURL == "" {
		return nil, fmt.Errorf("OP_LOGIN_PAGE_URL is required")
	}

	// issuer URL の末尾スラッシュを除去 (OIDC Discovery 1.0 Section 4.1)
	cfg.IssuerBaseURL = strings.TrimRight(cfg.IssuerBaseURL, "/")

	return cfg, nil
}

func (c *Config) IsSecure() bool {
	return strings.HasPrefix(c.IssuerBaseURL, "https://")
}
