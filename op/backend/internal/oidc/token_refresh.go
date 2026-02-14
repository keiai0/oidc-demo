package oidc

import (
	"context"
	"fmt"
	"time"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// RefreshTokenGrantInput はリフレッシュトークングラントの入力
type RefreshTokenGrantInput struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	Scope        string
}

// handleRefreshTokenGrantLogic はリフレッシュトークングラントのビジネスロジック。
// Refresh Token Rotation + Reuse Detection (RFC 9700) を実装。
func (h *TokenHandler) handleRefreshTokenGrantLogic(ctx context.Context, input *RefreshTokenGrantInput) (*TokenResponse, error) {
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

	if !client.HasGrantType("refresh_token") {
		return nil, ErrUnsupportedGrantType
	}

	// リフレッシュトークン検証 (SHA-256ハッシュで検索)
	tokenHash := h.sha256Hex(input.RefreshToken)
	rt, err := h.refreshTokenStore.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("failed to find refresh token: %w", err)
	}
	if rt == nil {
		return nil, ErrInvalidGrant
	}

	// Reuse Detection: 既に失効済みのリフレッシュトークンが使われた場合
	// → 盗まれた可能性があるためセッション全体のトークンを失効
	if rt.RevokedAt != nil {
		_ = h.refreshTokenStore.MarkReuseDetected(ctx, rt.ID)
		_ = h.accessTokenStore.RevokeBySessionID(ctx, rt.SessionID)
		_ = h.refreshTokenStore.RevokeBySessionID(ctx, rt.SessionID)
		return nil, ErrInvalidGrant
	}

	// 有効期限チェック
	if rt.ExpiresAt.Before(time.Now()) {
		return nil, ErrInvalidGrant
	}

	// セッション有効性チェック
	if rt.Session.RevokedAt != nil || rt.Session.ExpiresAt.Before(time.Now()) {
		return nil, ErrInvalidGrant
	}

	// 古いリフレッシュトークンを失効 (Rotation)
	if err := h.refreshTokenStore.Revoke(ctx, rt.ID); err != nil {
		return nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
	}

	// 古いアクセストークンを失効
	_ = h.accessTokenStore.Revoke(ctx, rt.AccessTokenID)

	// テナント情報取得
	tenant, err := h.tenantFinder.FindByID(ctx, rt.Session.TenantID)
	if err != nil || tenant == nil {
		return nil, fmt.Errorf("failed to find tenant: %w", err)
	}

	issuer := h.issuerBaseURL + "/" + tenant.Code
	userID := rt.Session.UserID.String()

	// スコープ: リクエストのscopeが指定されていればそれを使う（ただし元のスコープ以下）
	scope := rt.AccessToken.Scope
	if input.Scope != "" {
		scope = input.Scope
	}

	// 新しいアクセストークン生成
	accessTokenLifetime := time.Duration(tenant.AccessTokenLifetime) * time.Second
	accessJTI, accessTokenStr, err := h.tokenSigner.SignAccessToken(ctx, &model.AccessTokenClaims{
		Issuer:    issuer,
		Subject:   userID,
		Audience:  client.ClientID,
		Scope:     scope,
		SessionID: rt.SessionID.String(),
	}, accessTokenLifetime)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	accessToken := &model.AccessToken{
		JTI:       accessJTI,
		SessionID: rt.SessionID,
		ClientID:  client.ID,
		Scope:     scope,
		ExpiresAt: time.Now().Add(accessTokenLifetime),
	}
	if err := h.accessTokenStore.Create(ctx, accessToken); err != nil {
		return nil, fmt.Errorf("failed to save access token: %w", err)
	}

	// 新しいリフレッシュトークン生成 (Rotation)
	newRefreshTokenStr, newTokenHash, err := h.tokenSigner.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	refreshTokenLifetime := time.Duration(tenant.RefreshTokenLifetime) * time.Second
	newRefreshToken := &model.RefreshToken{
		TokenHash:     newTokenHash,
		ParentID:      &rt.ID,
		SessionID:     rt.SessionID,
		AccessTokenID: accessToken.ID,
		ExpiresAt:     time.Now().Add(refreshTokenLifetime),
	}
	if err := h.refreshTokenStore.Create(ctx, newRefreshToken); err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessTokenStr,
		TokenType:    "Bearer",
		ExpiresIn:    tenant.AccessTokenLifetime,
		RefreshToken: newRefreshTokenStr,
		Scope:        scope,
	}, nil
}
