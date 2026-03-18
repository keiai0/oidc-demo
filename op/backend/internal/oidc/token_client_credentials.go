package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/audit"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/metrics"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// ClientCredentialsGrantInput は Client Credentials Grant の入力。
// 仕様参照: RFC 6749 Section 4.4
type ClientCredentialsGrantInput struct {
	ClientID     string
	ClientSecret string
	Scope        string
	DPoPJKT      string
	TenantCode   string
}

func (h *TokenHandler) handleClientCredentialsGrant(c echo.Context, dpopJKT string) error {
	clientID, clientSecret := extractClientCredentials(c)
	if clientID == "" || clientSecret == "" {
		return tokenError(c, http.StatusUnauthorized, "invalid_client", "client credentials required")
	}

	tenantCode := c.Param("tenant_code")
	scope := c.FormValue("scope")

	resp, err := h.handleClientCredentialsGrantLogic(c.Request().Context(), &ClientCredentialsGrantInput{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scope:        scope,
		DPoPJKT:      dpopJKT,
		TenantCode:   tenantCode,
	})
	if err != nil {
		if errors.Is(err, ErrInvalidClient) {
			return tokenError(c, http.StatusUnauthorized, "invalid_client", "")
		}
		if errors.Is(err, ErrUnsupportedGrantType) {
			return tokenError(c, http.StatusBadRequest, "unsupported_grant_type", "")
		}
		if errors.Is(err, ErrInvalidScope) {
			return tokenError(c, http.StatusBadRequest, "invalid_scope", "")
		}
		h.logger.ErrorContext(c.Request().Context(), "client credentials error", "error", err)
		return tokenError(c, http.StatusInternalServerError, "server_error", "")
	}

	h.audit.LogEvent(c.Request().Context(), audit.EventTokenIssued,
		audit.ClientAttr(clientID), audit.GrantTypeAttr("client_credentials"), audit.ResultAttr("success"),
	)
	metrics.TokenIssuedTotal.WithLabelValues("client_credentials").Inc()

	return c.JSON(http.StatusOK, resp)
}

// handleClientCredentialsGrantLogic は Client Credentials Grant のビジネスロジック。
// RFC 6749 Section 4.4: クライアント認証のみでアクセストークンを発行する。
// ID トークンは発行しない（ユーザー認証ではない）。
// リフレッシュトークンは発行しない (RFC 6749 Section 4.4.3)。
func (h *TokenHandler) handleClientCredentialsGrantLogic(ctx context.Context, input *ClientCredentialsGrantInput) (*TokenResponse, error) {
	// クライアント認証
	client, err := h.clientFinder.FindByClientID(ctx, input.ClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to find client: %w", err)
	}
	if client == nil || client.Status != "active" {
		return nil, ErrInvalidClient
	}

	match, err := h.verifyPassword(input.ClientSecret, client.ClientSecretHash)
	if err != nil || !match {
		return nil, ErrInvalidClient
	}

	// grant_type サポート確認
	if !client.HasGrantType("client_credentials") {
		return nil, ErrUnsupportedGrantType
	}

	// テナント解決
	tenant, err := h.tenantFinder.FindByCode(ctx, input.TenantCode)
	if err != nil {
		return nil, fmt.Errorf("failed to find tenant: %w", err)
	}
	if tenant == nil {
		return nil, ErrInvalidClient
	}

	// テナント-クライアント紐づき検証
	belongs, err := h.tenantClientChecker.ExistsByTenantAndClient(ctx, tenant.ID, client.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check tenant-client association: %w", err)
	}
	if !belongs {
		return nil, ErrInvalidClient
	}

	issuer := h.issuerBaseURL + "/" + tenant.Code

	// アクセストークン生成
	// sub はクライアント自身 (M2M: ユーザーなし)
	accessTokenLifetime := time.Duration(tenant.AccessTokenLifetime) * time.Second
	atClaims := &model.AccessTokenClaims{
		Issuer:   issuer,
		Subject:  client.ClientID,
		Audience: client.ClientID,
		Scope:    input.Scope,
		// SessionID は空（セッションなし）
	}
	if input.DPoPJKT != "" {
		atClaims.Confirmation = &model.TokenConfirmation{JKT: input.DPoPJKT}
	}

	accessJTI, accessTokenStr, err := h.tokenSigner.SignAccessToken(ctx, atClaims, accessTokenLifetime)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// DB 保存 (SessionID = nil)
	accessToken := &model.AccessToken{
		JTI:       accessJTI,
		SessionID: nil,
		ClientID:  client.ID,
		Scope:     input.Scope,
		ExpiresAt: time.Now().Add(accessTokenLifetime),
	}
	if input.DPoPJKT != "" {
		accessToken.DPoPJKT = &input.DPoPJKT
	}
	if err := h.accessTokenStore.Create(ctx, accessToken); err != nil {
		return nil, fmt.Errorf("failed to save access token: %w", err)
	}

	tokenType := "Bearer"
	if input.DPoPJKT != "" {
		tokenType = "DPoP"
	}

	return &TokenResponse{
		AccessToken: accessTokenStr,
		TokenType:   tokenType,
		ExpiresIn:   tenant.AccessTokenLifetime,
		Scope:       input.Scope,
		// IDToken: 発行しない (RFC 6749 Section 4.4)
		// RefreshToken: 発行しない (RFC 6749 Section 4.4.3)
	}, nil
}
