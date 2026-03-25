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
	DPoPJKT      string // DPoP JWK Thumbprint (空の場合は Bearer)
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
	// クライアント認証（Pairwise sub の sector identifier 解決のため redirect_uri もロード）
	client, err := h.clientFinder.FindByClientIDWithRedirectURIs(ctx, input.ClientID)
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

	// PKCE 検証（デモモード時はスキップ）
	if !h.demoMode && authCode.CodeChallenge != nil && *authCode.CodeChallenge != "" {
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
	// Pairwise Subject Identifier (OIDC Core Section 8): クライアント設定に応じて sub を変換
	userID := ResolveSubject(client, authCode.Session.UserID.String())

	// claims リクエストパラメータの解析 (OIDC Core 1.0 Section 5.5)
	var claimsReq *model.ClaimsRequest
	if authCode.ClaimsParam != nil {
		var err error
		claimsReq, err = ParseClaimsRequest(*authCode.ClaimsParam)
		if err != nil {
			return nil, fmt.Errorf("failed to parse claims parameter: %w", err)
		}
	}

	// アクセストークン生成
	accessTokenLifetime := time.Duration(tenant.AccessTokenLifetime) * time.Second
	atClaims := &model.AccessTokenClaims{
		Issuer:    issuer,
		Subject:   userID,
		Audience:  client.ClientID,
		Scope:     authCode.Scope,
		SessionID: authCode.SessionID.String(),
	}
	// DPoP: cnf.jkt クレームを含める (RFC 9449 Section 6.1)
	if input.DPoPJKT != "" {
		atClaims.Confirmation = &model.TokenConfirmation{JKT: input.DPoPJKT}
	}
	accessJTI, accessTokenStr, err := h.tokenSigner.SignAccessToken(ctx, atClaims, accessTokenLifetime)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// アクセストークンDB保存（claims_param を userinfo で使用するためコピー）
	accessToken := &model.AccessToken{
		JTI:         accessJTI,
		SessionID:   &authCode.SessionID,
		ClientID:    client.ID,
		Scope:       authCode.Scope,
		ClaimsParam: authCode.ClaimsParam,
		ExpiresAt:   time.Now().Add(accessTokenLifetime),
	}
	if input.DPoPJKT != "" {
		accessToken.DPoPJKT = &input.DPoPJKT
	}
	if err := h.accessTokenStore.Create(ctx, accessToken); err != nil {
		return nil, fmt.Errorf("failed to save access token: %w", err)
	}

	// IDトークン生成 (at_hash 含む)
	atHash := h.computeATHash(accessTokenStr)
	idTokenLifetime := time.Duration(tenant.IDTokenLifetime) * time.Second

	// claims パラメータで要求された id_token 向け追加クレームを解決 (OIDC Core Section 5.5)
	// ユーザー由来のクレーム (name, email 等) も含めてベストエフォートで ID トークンに含める。
	var user *model.User
	if claimsReq != nil && claimsReq.IDToken != nil {
		user, _ = h.userFinder.FindByID(ctx, authCode.Session.UserID)
	}
	extraClaims := ResolveIDTokenClaims(claimsReq, user, &authCode.Session)

	idTokenJTI, idTokenStr, err := h.tokenSigner.SignIDToken(ctx, &model.IDTokenClaims{
		Issuer:      issuer,
		Subject:     userID,
		Audience:    client.ClientID,
		Nonce:       authCode.Nonce,
		AuthTime:    authCode.Session.AuthTime,
		ATHash:      atHash,
		ACR:         authCode.Session.ACR,
		AMR:         []string(authCode.Session.AMR),
		SessionID:   authCode.SessionID.String(),
		ExtraClaims: extraClaims,
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

		now := time.Now()

		// 絶対有効期限: クライアント設定 > テナント設定
		absoluteLifetimeSec := tenant.RefreshTokenLifetime
		if client.RefreshTokenLifetime != nil {
			absoluteLifetimeSec = *client.RefreshTokenLifetime
		}
		absoluteExpiresAt := now.Add(time.Duration(absoluteLifetimeSec) * time.Second)

		// スライディング有効期限: クライアントに idle_timeout 設定がある場合
		expiresAt := absoluteExpiresAt
		if client.RefreshTokenIdleTimeout != nil {
			idleExpiresAt := now.Add(time.Duration(*client.RefreshTokenIdleTimeout) * time.Second)
			if idleExpiresAt.Before(absoluteExpiresAt) {
				expiresAt = idleExpiresAt
			}
		}

		refreshToken := &model.RefreshToken{
			TokenHash:         tokenHash,
			SessionID:         authCode.SessionID,
			AccessTokenID:     accessToken.ID,
			ExpiresAt:         expiresAt,
			AbsoluteExpiresAt: &absoluteExpiresAt,
		}
		if err := h.refreshTokenStore.Create(ctx, refreshToken); err != nil {
			return nil, fmt.Errorf("failed to save refresh token: %w", err)
		}
	}

	tokenType := "Bearer"
	if input.DPoPJKT != "" {
		tokenType = "DPoP"
	}

	return &TokenResponse{
		AccessToken:  accessTokenStr,
		TokenType:    tokenType,
		ExpiresIn:    tenant.AccessTokenLifetime,
		RefreshToken: refreshTokenStr,
		IDToken:      idTokenStr,
		Scope:        authCode.Scope,
	}, nil
}
