package oidc

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
)

type RevokeHandler struct {
	clientFinder      ClientFinder
	accessTokenStore  AccessTokenStore
	refreshTokenStore RefreshTokenStore
	tokenValidator    TokenValidator
	verifyPassword    VerifyPasswordFunc
	sha256Hex         SHA256HexFunc
}

func NewRevokeHandler(
	clientFinder ClientFinder,
	accessTokenStore AccessTokenStore,
	refreshTokenStore RefreshTokenStore,
	tokenValidator TokenValidator,
	verifyPassword VerifyPasswordFunc,
	sha256Hex SHA256HexFunc,
) *RevokeHandler {
	return &RevokeHandler{
		clientFinder:      clientFinder,
		accessTokenStore:  accessTokenStore,
		refreshTokenStore: refreshTokenStore,
		tokenValidator:    tokenValidator,
		verifyPassword:    verifyPassword,
		sha256Hex:         sha256Hex,
	}
}

// Handle は POST /{tenant_code}/revoke を処理する
// 仕様参照: RFC 7009 Section 2
func (h *RevokeHandler) Handle(c echo.Context) error {
	// クライアント認証
	clientID, clientSecret := extractClientCredentials(c)
	if clientID == "" || clientSecret == "" {
		return tokenError(c, http.StatusUnauthorized, "invalid_client", "client credentials required")
	}

	client, err := h.clientFinder.FindByClientID(c.Request().Context(), clientID)
	if err != nil || client == nil || client.Status != "active" {
		return tokenError(c, http.StatusUnauthorized, "invalid_client", "")
	}

	match, err := h.verifyPassword(clientSecret, client.ClientSecretHash)
	if err != nil || !match {
		return tokenError(c, http.StatusUnauthorized, "invalid_client", "")
	}

	token := c.FormValue("token")
	if token == "" {
		return c.NoContent(http.StatusOK)
	}

	tokenTypeHint := c.FormValue("token_type_hint")
	ctx := c.Request().Context()

	switch tokenTypeHint {
	case "refresh_token":
		if !h.revokeRefreshToken(ctx, token) {
			h.revokeAccessToken(ctx, token)
		}
	case "access_token":
		if !h.revokeAccessToken(ctx, token) {
			h.revokeRefreshToken(ctx, token)
		}
	default:
		if !h.revokeAccessToken(ctx, token) {
			h.revokeRefreshToken(ctx, token)
		}
	}

	// RFC 7009 Section 2.2: 存在しないトークンでも 200 OK (MUST)
	return c.NoContent(http.StatusOK)
}

func (h *RevokeHandler) revokeAccessToken(ctx context.Context, tokenStr string) bool {
	result, err := h.tokenValidator.ValidateAccessToken(ctx, tokenStr)
	if err != nil {
		return false
	}

	dbToken, err := h.accessTokenStore.FindByJTI(ctx, result.JTI)
	if err != nil || dbToken == nil {
		return false
	}

	_ = h.accessTokenStore.Revoke(ctx, dbToken.ID)
	return true
}

func (h *RevokeHandler) revokeRefreshToken(ctx context.Context, tokenStr string) bool {
	tokenHash := h.sha256Hex(tokenStr)

	rt, err := h.refreshTokenStore.FindByTokenHash(ctx, tokenHash)
	if err != nil || rt == nil {
		return false
	}

	_ = h.refreshTokenStore.Revoke(ctx, rt.ID)
	_ = h.accessTokenStore.Revoke(ctx, rt.AccessTokenID)
	return true
}
