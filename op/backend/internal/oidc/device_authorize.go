package oidc

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/audit"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

const (
	// deviceCodeBytes は device_code の生成に使用するランダムバイト数。
	// RFC 8628 Section 6.1: 高エントロピーを要求。
	deviceCodeBytes = 32

	// deviceCodeLifetime は device_code のデフォルト有効期間。
	// RFC 8628 Section 3.2: expires_in で通知する。
	deviceCodeLifetime = 600 * time.Second

	// defaultPollInterval はデバイスのデフォルトポーリング間隔（秒）。
	// RFC 8628 Section 3.2: interval パラメータ。
	defaultPollInterval = 5
)

// userCodeChars は user_code に使用する文字セット。
// 紛らわしい文字（0/O, 1/I/L）を除外した大文字子音 + 数字。
// RFC 8628 Section 6.1: ユーザーが入力しやすい短いコード。
var userCodeChars = []byte("BCDFGHJKLMNPQRSTVWXZ")

// DeviceAuthorizeHandler は POST /{tenant_code}/device/authorize を処理する。
// 仕様参照: RFC 8628 Section 3.1, 3.2
type DeviceAuthorizeHandler struct {
	clientFinder        ClientFinder
	tenantFinder        TenantFinder
	tenantClientChecker TenantClientChecker
	deviceAuthStore     DeviceAuthorizationRequestStore
	verifyPassword      VerifyPasswordFunc
	audit               *audit.AuditLogger
	logger              *slog.Logger
	issuerBaseURL       string
	frontendBaseURL     string
}

// NewDeviceAuthorizeHandler は DeviceAuthorizeHandler を生成する。
func NewDeviceAuthorizeHandler(
	clientFinder ClientFinder,
	tenantFinder TenantFinder,
	tenantClientChecker TenantClientChecker,
	deviceAuthStore DeviceAuthorizationRequestStore,
	verifyPassword VerifyPasswordFunc,
	auditLog *audit.AuditLogger,
	logger *slog.Logger,
	issuerBaseURL string,
	frontendBaseURL string,
) *DeviceAuthorizeHandler {
	return &DeviceAuthorizeHandler{
		clientFinder:        clientFinder,
		tenantFinder:        tenantFinder,
		tenantClientChecker: tenantClientChecker,
		deviceAuthStore:     deviceAuthStore,
		verifyPassword:      verifyPassword,
		audit:               auditLog,
		logger:              logger,
		issuerBaseURL:       issuerBaseURL,
		frontendBaseURL:     frontendBaseURL,
	}
}

// Handle は POST /{tenant_code}/device/authorize を処理する。
// 仕様参照: RFC 8628 Section 3.1 (Device Authorization Request), 3.2 (Device Authorization Response)
func (h *DeviceAuthorizeHandler) Handle(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Pragma", "no-cache")

	ctx := c.Request().Context()
	tenantCode := c.Param("tenant_code")

	// テナント検証
	tenant, err := h.tenantFinder.FindByCode(ctx, tenantCode)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to find tenant", "error", err)
		return tokenError(c, http.StatusInternalServerError, "server_error", "")
	}
	if tenant == nil {
		return tokenError(c, http.StatusBadRequest, "invalid_request", "unknown tenant")
	}

	// クライアント認証
	// RFC 8628 Section 3.1: Confidential clients MUST authenticate.
	clientID, clientSecret := extractClientCredentials(c)
	if clientID == "" {
		// client_id は Form からも取得可能（public client 用）
		clientID = c.FormValue("client_id")
	}
	if clientID == "" {
		return tokenError(c, http.StatusUnauthorized, "invalid_client", "client_id is required")
	}

	client, err := h.clientFinder.FindByClientID(ctx, clientID)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to find client", "error", err)
		return tokenError(c, http.StatusInternalServerError, "server_error", "")
	}
	if client == nil || client.Status != "active" {
		return tokenError(c, http.StatusUnauthorized, "invalid_client", "")
	}

	// Confidential client は secret 必須
	if clientSecret != "" {
		match, err := h.verifyPassword(clientSecret, client.ClientSecretHash)
		if err != nil || !match {
			return tokenError(c, http.StatusUnauthorized, "invalid_client", "")
		}
	}

	// grant_type サポート確認
	if !client.HasGrantType("urn:ietf:params:oauth:grant-type:device_code") {
		return tokenError(c, http.StatusBadRequest, "unauthorized_client", "device_code grant not allowed for this client")
	}

	// テナント-クライアント紐づき検証
	belongs, err := h.tenantClientChecker.ExistsByTenantAndClient(ctx, tenant.ID, client.ID)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to check tenant-client", "error", err)
		return tokenError(c, http.StatusInternalServerError, "server_error", "")
	}
	if !belongs {
		return tokenError(c, http.StatusBadRequest, "invalid_request", "client does not belong to this tenant")
	}

	scope := c.FormValue("scope")

	// device_code 生成: crypto/rand 32 bytes → base64url
	deviceCode, err := generateDeviceCode()
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to generate device code", "error", err)
		return tokenError(c, http.StatusInternalServerError, "server_error", "")
	}

	// user_code 生成: XXXX-XXXX 形式
	userCode, err := generateUserCode()
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to generate user code", "error", err)
		return tokenError(c, http.StatusInternalServerError, "server_error", "")
	}

	req := &model.DeviceAuthorizationRequest{
		TenantID:     tenant.ID,
		ClientID:     client.ID,
		DeviceCode:   deviceCode,
		UserCode:     userCode,
		Scope:        scope,
		Status:       "pending",
		PollInterval: defaultPollInterval,
		ExpiresAt:    time.Now().Add(deviceCodeLifetime),
	}

	if err := h.deviceAuthStore.Create(ctx, req); err != nil {
		h.logger.ErrorContext(ctx, "failed to store device auth request", "error", err)
		return tokenError(c, http.StatusInternalServerError, "server_error", "")
	}

	h.audit.LogEvent(ctx, audit.EventDeviceAuthorization,
		audit.ClientAttr(clientID), audit.TenantAttr(tenantCode),
	)

	// verification_uri はユーザーがブラウザでアクセスする URL のため、フロントエンド URL を使用
	verificationURI := h.frontendBaseURL + "/device"

	// RFC 8628 Section 3.2: Device Authorization Response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"device_code":               deviceCode,
		"user_code":                 userCode,
		"verification_uri":          verificationURI,
		"verification_uri_complete": verificationURI + "?user_code=" + userCode,
		"expires_in":                int(deviceCodeLifetime.Seconds()),
		"interval":                  defaultPollInterval,
	})
}

// generateDeviceCode は暗号学的に安全な device_code を生成する。
func generateDeviceCode() (string, error) {
	b := make([]byte, deviceCodeBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// generateUserCode は XXXX-XXXX 形式のユーザーコードを生成する。
// 紛らわしい文字を除外し、入力しやすさを優先する。
func generateUserCode() (string, error) {
	code := make([]byte, 9) // 8文字 + ハイフン
	for i := 0; i < 9; i++ {
		if i == 4 {
			code[i] = '-'
			continue
		}
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(userCodeChars))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random index: %w", err)
		}
		code[i] = userCodeChars[idx.Int64()]
	}
	return string(code), nil
}
