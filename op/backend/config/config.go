package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port              string
	DSN               string
	BaseURL           string
	KeyEncryptionKey  string
	FrontendBaseURL   string
	AllowedRPOrigins  []string
	WebAuthnRPID      string
	WebAuthnRPName    string
	WebAuthnRPOrigins []string
	// 署名鍵自動ローテーション設定
	KeyRotationIntervalDays int // ローテーション間隔（日）。デフォルト 90 日
	KeyGracePeriodDays      int // passive 鍵の猶予期間（日）。デフォルト 7 日
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

	// RP オリジン（CORS 用、カンマ区切り）
	if rpOrigins := os.Getenv("OP_ALLOWED_RP_ORIGINS"); rpOrigins != "" {
		for _, origin := range strings.Split(rpOrigins, ",") {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				cfg.AllowedRPOrigins = append(cfg.AllowedRPOrigins, origin)
			}
		}
	}

	// WebAuthn RP 設定: 環境変数があればそちらを優先、なければ FrontendBaseURL から導出
	cfg.WebAuthnRPID = os.Getenv("OP_WEBAUTHN_RP_ID")
	if cfg.WebAuthnRPID == "" {
		if parsed, err := url.Parse(cfg.FrontendBaseURL); err == nil {
			cfg.WebAuthnRPID = parsed.Hostname()
		}
	}

	cfg.WebAuthnRPName = os.Getenv("OP_WEBAUTHN_RP_NAME")
	if cfg.WebAuthnRPName == "" {
		cfg.WebAuthnRPName = "OIDC Demo"
	}

	cfg.WebAuthnRPOrigins = []string{strings.TrimRight(cfg.FrontendBaseURL, "/")}

	// 署名鍵ローテーション設定（省略時はデフォルト値）
	cfg.KeyRotationIntervalDays = parseIntEnv("OP_KEY_ROTATION_INTERVAL_DAYS", 90)
	cfg.KeyGracePeriodDays = parseIntEnv("OP_KEY_GRACE_PERIOD_DAYS", 7)

	return cfg, nil
}

func (c *Config) IsSecure() bool {
	return strings.HasPrefix(c.BaseURL, "https://")
}

func parseIntEnv(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return defaultVal
	}
	return n
}
