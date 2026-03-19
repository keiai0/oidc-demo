package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/audit"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/metrics"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// DeviceCodeGrantInput は Device Code Grant の入力。
// 仕様参照: RFC 8628 Section 3.4
type DeviceCodeGrantInput struct {
	ClientID     string
	ClientSecret string
	DeviceCode   string
	DPoPJKT      string
	TenantCode   string
}

const deviceCodeGrantType = "urn:ietf:params:oauth:grant-type:device_code"

func (h *TokenHandler) handleDeviceCodeGrant(c echo.Context, dpopJKT string) error {
	clientID, clientSecret := extractClientCredentials(c)
	if clientID == "" {
		clientID = c.FormValue("client_id")
	}
	if clientID == "" {
		return tokenError(c, http.StatusUnauthorized, "invalid_client", "client_id is required")
	}

	deviceCode := c.FormValue("device_code")
	if deviceCode == "" {
		return tokenError(c, http.StatusBadRequest, "invalid_request", "device_code is required")
	}

	tenantCode := c.Param("tenant_code")

	resp, err := h.handleDeviceCodeGrantLogic(c.Request().Context(), &DeviceCodeGrantInput{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		DeviceCode:   deviceCode,
		DPoPJKT:      dpopJKT,
		TenantCode:   tenantCode,
	})
	if err != nil {
		if errors.Is(err, ErrInvalidClient) {
			return tokenError(c, http.StatusUnauthorized, "invalid_client", "")
		}
		if errors.Is(err, ErrAuthorizationPending) {
			return tokenError(c, http.StatusBadRequest, "authorization_pending", "")
		}
		if errors.Is(err, ErrSlowDown) {
			return tokenError(c, http.StatusBadRequest, "slow_down", "")
		}
		if errors.Is(err, ErrExpiredToken) {
			return tokenError(c, http.StatusBadRequest, "expired_token", "")
		}
		if errors.Is(err, ErrAccessDenied) {
			return tokenError(c, http.StatusBadRequest, "access_denied", "")
		}
		if errors.Is(err, ErrInvalidGrant) {
			return tokenError(c, http.StatusBadRequest, "invalid_grant", "")
		}
		h.logger.ErrorContext(c.Request().Context(), "device code grant error", "error", err)
		return tokenError(c, http.StatusInternalServerError, "server_error", "")
	}

	h.audit.LogEvent(c.Request().Context(), audit.EventTokenIssued,
		audit.ClientAttr(clientID), audit.GrantTypeAttr(deviceCodeGrantType), audit.ResultAttr("success"),
	)
	metrics.TokenIssuedTotal.WithLabelValues("device_code").Inc()

	return c.JSON(http.StatusOK, resp)
}

// handleDeviceCodeGrantLogic は Device Code Grant のビジネスロジック。
// RFC 8628 Section 3.4: Token Request, Section 3.5: Token Response
func (h *TokenHandler) handleDeviceCodeGrantLogic(ctx context.Context, input *DeviceCodeGrantInput) (*TokenResponse, error) {
	// クライアント認証
	client, err := h.clientFinder.FindByClientID(ctx, input.ClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to find client: %w", err)
	}
	if client == nil || client.Status != "active" {
		return nil, ErrInvalidClient
	}

	// Confidential client は secret 検証必須
	if input.ClientSecret != "" {
		match, err := h.verifyPassword(input.ClientSecret, client.ClientSecretHash)
		if err != nil || !match {
			return nil, ErrInvalidClient
		}
	}

	if !client.HasGrantType(deviceCodeGrantType) {
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

	belongs, err := h.tenantClientChecker.ExistsByTenantAndClient(ctx, tenant.ID, client.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check tenant-client: %w", err)
	}
	if !belongs {
		return nil, ErrInvalidClient
	}

	// device_code 検索
	deviceReq, err := h.deviceAuthStore.FindByDeviceCode(ctx, input.DeviceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to find device auth request: %w", err)
	}
	if deviceReq == nil {
		return nil, ErrInvalidGrant
	}

	// クライアント一致確認
	if deviceReq.ClientID != client.ID {
		return nil, ErrInvalidGrant
	}

	// 有効期限チェック (RFC 8628 Section 3.5)
	if deviceReq.IsExpired() {
		return nil, ErrExpiredToken
	}

	// ポーリング間隔チェック (RFC 8628 Section 3.5)
	now := time.Now()
	if deviceReq.LastPolledAt != nil {
		elapsed := now.Sub(*deviceReq.LastPolledAt)
		if elapsed < time.Duration(deviceReq.PollInterval)*time.Second {
			// slow_down: interval を +5s する (RFC 8628 Section 3.5)
			if err := h.deviceAuthStore.IncrementPollInterval(ctx, deviceReq.ID, 5); err != nil {
				return nil, fmt.Errorf("failed to increment poll interval: %w", err)
			}
			// last_polled_at も更新
			if err := h.deviceAuthStore.UpdateLastPolledAt(ctx, deviceReq.ID, now); err != nil {
				return nil, fmt.Errorf("failed to update last_polled_at: %w", err)
			}
			return nil, ErrSlowDown
		}
	}

	// last_polled_at 更新
	if err := h.deviceAuthStore.UpdateLastPolledAt(ctx, deviceReq.ID, now); err != nil {
		return nil, fmt.Errorf("failed to update last_polled_at: %w", err)
	}

	// ステータス判定
	switch {
	case deviceReq.IsPending():
		return nil, ErrAuthorizationPending
	case deviceReq.IsDenied():
		return nil, ErrAccessDenied
	case !deviceReq.IsApproved():
		return nil, ErrInvalidGrant
	}

	// approved: session_id からユーザー情報を取得してトークン発行
	if deviceReq.SessionID == nil {
		return nil, fmt.Errorf("approved device request has no session_id")
	}

	session, err := h.sessionFinder.FindByID(ctx, *deviceReq.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find session: %w", err)
	}
	if session == nil || !session.IsValid() {
		return nil, ErrInvalidGrant
	}

	user, err := h.userFinder.FindByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidGrant
	}

	issuer := h.issuerBaseURL + "/" + tenant.Code

	// Subject 決定（pairwise / public）
	sub := ResolveSubject(client, user.ID.String())

	// アクセストークン生成
	accessTokenLifetime := time.Duration(tenant.AccessTokenLifetime) * time.Second
	atClaims := &model.AccessTokenClaims{
		Issuer:    issuer,
		Subject:   sub,
		Audience:  client.ClientID,
		Scope:     deviceReq.Scope,
		SessionID: deviceReq.SessionID.String(),
	}
	if input.DPoPJKT != "" {
		atClaims.Confirmation = &model.TokenConfirmation{JKT: input.DPoPJKT}
	}

	accessJTI, accessTokenStr, err := h.tokenSigner.SignAccessToken(ctx, atClaims, accessTokenLifetime)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	accessToken := &model.AccessToken{
		JTI:       accessJTI,
		SessionID: deviceReq.SessionID,
		ClientID:  client.ID,
		Scope:     deviceReq.Scope,
		ExpiresAt: now.Add(accessTokenLifetime),
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

	resp := &TokenResponse{
		AccessToken: accessTokenStr,
		TokenType:   tokenType,
		ExpiresIn:   tenant.AccessTokenLifetime,
		Scope:       deviceReq.Scope,
	}

	// ID Token 発行（openid スコープがある場合）
	scopes := strings.Split(deviceReq.Scope, " ")
	if containsScope(scopes, "openid") {
		idTokenLifetime := time.Duration(tenant.IDTokenLifetime) * time.Second
		idClaims := &model.IDTokenClaims{
			Issuer:    issuer,
			Subject:   sub,
			Audience:  client.ClientID,
			AuthTime:  session.AuthTime,
			ACR:       session.ACR,
			AMR:       []string(session.AMR),
			ATHash:    h.computeATHash(accessTokenStr),
			SessionID: session.ID.String(),
		}

		idJTI, idTokenStr, err := h.tokenSigner.SignIDToken(ctx, idClaims, idTokenLifetime)
		if err != nil {
			return nil, fmt.Errorf("failed to sign id token: %w", err)
		}

		idToken := &model.IDToken{
			JTI:       idJTI,
			SessionID: *deviceReq.SessionID,
			ClientID:  client.ID,
			ExpiresAt: now.Add(idTokenLifetime),
		}
		if err := h.idTokenCreator.Create(ctx, idToken); err != nil {
			return nil, fmt.Errorf("failed to save id token: %w", err)
		}

		resp.IDToken = idTokenStr
	}

	// Refresh Token 発行（offline_access スコープ + クライアントが refresh_token grant 対応）
	if containsScope(scopes, "offline_access") && client.HasGrantType("refresh_token") {
		refreshTokenStr, refreshTokenHash, err := h.tokenSigner.GenerateRefreshToken()
		if err != nil {
			return nil, fmt.Errorf("failed to generate refresh token: %w", err)
		}

		// 絶対有効期限: クライアント設定 > テナント設定
		absoluteLifetimeSec := tenant.RefreshTokenLifetime
		if client.RefreshTokenLifetime != nil {
			absoluteLifetimeSec = *client.RefreshTokenLifetime
		}
		absoluteExpiresAt := now.Add(time.Duration(absoluteLifetimeSec) * time.Second)

		// スライディング有効期限
		expiresAt := absoluteExpiresAt
		if client.RefreshTokenIdleTimeout != nil {
			idleExpiresAt := now.Add(time.Duration(*client.RefreshTokenIdleTimeout) * time.Second)
			if idleExpiresAt.Before(absoluteExpiresAt) {
				expiresAt = idleExpiresAt
			}
		}

		rt := &model.RefreshToken{
			TokenHash:         refreshTokenHash,
			SessionID:         *deviceReq.SessionID,
			AccessTokenID:     accessToken.ID,
			ExpiresAt:         expiresAt,
			AbsoluteExpiresAt: &absoluteExpiresAt,
		}
		if err := h.refreshTokenStore.Create(ctx, rt); err != nil {
			return nil, fmt.Errorf("failed to save refresh token: %w", err)
		}

		resp.RefreshToken = refreshTokenStr
	}

	return resp, nil
}
