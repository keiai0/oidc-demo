package oidc

import (
	"context"
	"fmt"
	"time"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// AuthCodeGrantInput は認可コードグラントの入力
type AuthCodeGrantInput struct {
	ClientID     string
	ClientSecret string
	Code         string
	RedirectURI  string
	CodeVerifier string
}

// TokenResponse はトークンレスポンス
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope"`
}

// handleAuthCodeGrantLogic は認可コードグラントのビジネスロジック
func (h *TokenHandler) handleAuthCodeGrantLogic(ctx context.Context, input *AuthCodeGrantInput) (*TokenResponse, error) {
	// クライアント認証
	client, err := h.clientFinder.FindByClientID(ctx, input.ClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to find client: %w", err)
	}
	if client == nil || client.Status != "active" {
		return nil, ErrInvalidClient
	}

	// client_secret 検証
	match, err := h.verifyPassword(input.ClientSecret, client.ClientSecretHash)
	if err != nil || !match {
		return nil, ErrInvalidClient
	}

	// 認可コード検証
	authCode, err := h.authCodeStore.FindByCode(ctx, input.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to find auth code: %w", err)
	}
	if authCode == nil {
		return nil, ErrInvalidGrant
	}

	// 認可コード二重使用チェック (MUST: RFC 6749 Section 4.1.2)
	if authCode.IsUsed() {
		// SHOULD: 既発行トークンを失効
		_ = h.accessTokenStore.RevokeBySessionID(ctx, authCode.SessionID)
		_ = h.refreshTokenStore.RevokeBySessionID(ctx, authCode.SessionID)
		return nil, ErrInvalidGrant
	}

	if authCode.IsExpired() {
		return nil, ErrInvalidGrant
	}

	// client_id 一致チェック
	if authCode.ClientID != client.ID {
		return nil, ErrInvalidGrant
	}

	// redirect_uri 一致チェック
	if authCode.RedirectURI != input.RedirectURI {
		return nil, ErrInvalidGrant
	}

	// PKCE 検証
	if authCode.CodeChallenge != nil && *authCode.CodeChallenge != "" {
		if input.CodeVerifier == "" {
			return nil, ErrInvalidGrant
		}
		if !h.verifyCodeChallenge(input.CodeVerifier, *authCode.CodeChallenge) {
			return nil, ErrInvalidGrant
		}
	}

	// 認可コードを使用済みにマーク
	if err := h.authCodeStore.MarkAsUsed(ctx, authCode.ID); err != nil {
		return nil, fmt.Errorf("failed to mark auth code as used: %w", err)
	}

	// テナント情報取得（トークン有効期限に使用）
	tenant, err := h.tenantFinder.FindByID(ctx, authCode.Session.TenantID)
	if err != nil || tenant == nil {
		return nil, fmt.Errorf("failed to find tenant: %w", err)
	}

	issuer := h.issuerBaseURL + "/" + tenant.Code
	userID := authCode.Session.UserID.String()

	// アクセストークン生成
	accessTokenLifetime := time.Duration(tenant.AccessTokenLifetime) * time.Second
	accessJTI, accessTokenStr, err := h.tokenSigner.SignAccessToken(ctx, &model.AccessTokenClaims{
		Issuer:    issuer,
		Subject:   userID,
		Audience:  client.ClientID,
		Scope:     authCode.Scope,
		SessionID: authCode.SessionID.String(),
	}, accessTokenLifetime)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// アクセストークンDB保存
	accessToken := &model.AccessToken{
		JTI:       accessJTI,
		SessionID: authCode.SessionID,
		ClientID:  client.ID,
		Scope:     authCode.Scope,
		ExpiresAt: time.Now().Add(accessTokenLifetime),
	}
	if err := h.accessTokenStore.Create(ctx, accessToken); err != nil {
		return nil, fmt.Errorf("failed to save access token: %w", err)
	}

	// IDトークン生成 (at_hash 含む)
	atHash := h.computeATHash(accessTokenStr)
	idTokenLifetime := time.Duration(tenant.IDTokenLifetime) * time.Second
	idTokenJTI, idTokenStr, err := h.tokenSigner.SignIDToken(ctx, &model.IDTokenClaims{
		Issuer:   issuer,
		Subject:  userID,
		Audience: client.ClientID,
		Nonce:    authCode.Nonce,
		AuthTime: authCode.Session.CreatedAt,
		ATHash:   atHash,
	}, idTokenLifetime)
	if err != nil {
		return nil, fmt.Errorf("failed to sign ID token: %w", err)
	}

	// IDトークンDB保存
	idToken := &model.IDToken{
		JTI:       idTokenJTI,
		SessionID: authCode.SessionID,
		ClientID:  client.ID,
		Nonce:     authCode.Nonce,
		ExpiresAt: time.Now().Add(idTokenLifetime),
	}
	if err := h.idTokenCreator.Create(ctx, idToken); err != nil {
		return nil, fmt.Errorf("failed to save ID token: %w", err)
	}

	// リフレッシュトークン生成 (offline_access スコープまたはrefresh_token grant対応時)
	var refreshTokenStr string
	if client.HasGrantType("refresh_token") {
		var tokenHash string
		refreshTokenStr, tokenHash, err = h.tokenSigner.GenerateRefreshToken()
		if err != nil {
			return nil, fmt.Errorf("failed to generate refresh token: %w", err)
		}

		refreshTokenLifetime := time.Duration(tenant.RefreshTokenLifetime) * time.Second
		refreshToken := &model.RefreshToken{
			TokenHash:     tokenHash,
			SessionID:     authCode.SessionID,
			AccessTokenID: accessToken.ID,
			ExpiresAt:     time.Now().Add(refreshTokenLifetime),
		}
		if err := h.refreshTokenStore.Create(ctx, refreshToken); err != nil {
			return nil, fmt.Errorf("failed to save refresh token: %w", err)
		}
	}

	return &TokenResponse{
		AccessToken:  accessTokenStr,
		TokenType:    "Bearer",
		ExpiresIn:    tenant.AccessTokenLifetime,
		RefreshToken: refreshTokenStr,
		IDToken:      idTokenStr,
		Scope:        authCode.Scope,
	}, nil
}
