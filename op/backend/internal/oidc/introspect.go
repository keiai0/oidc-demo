package oidc

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// IntrospectHandler は POST /{tenant_code}/introspect を処理する。
// 仕様参照: RFC 7662
type IntrospectHandler struct {
	clientFinder      ClientFinder
	accessTokenStore  AccessTokenStore
	refreshTokenStore RefreshTokenStore
	tokenValidator    TokenValidator
	userFinder        UserFinder
	verifyPassword    VerifyPasswordFunc
	sha256Hex         SHA256HexFunc
}

// NewIntrospectHandler は IntrospectHandler を生成する。
func NewIntrospectHandler(
	clientFinder ClientFinder,
	accessTokenStore AccessTokenStore,
	refreshTokenStore RefreshTokenStore,
	tokenValidator TokenValidator,
	userFinder UserFinder,
	verifyPassword VerifyPasswordFunc,
	sha256Hex SHA256HexFunc,
) *IntrospectHandler {
	return &IntrospectHandler{
		clientFinder:      clientFinder,
		accessTokenStore:  accessTokenStore,
		refreshTokenStore: refreshTokenStore,
		tokenValidator:    tokenValidator,
		userFinder:        userFinder,
		verifyPassword:    verifyPassword,
		sha256Hex:         sha256Hex,
	}
}

// Handle は POST /{tenant_code}/introspect を処理する。
// 仕様参照: RFC 7662 Section 2
func (h *IntrospectHandler) Handle(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Pragma", "no-cache")

	ctx := c.Request().Context()

	// クライアント認証 (MUST: RFC 7662 Section 2.1)
	clientID, clientSecret := extractClientCredentials(c)
	if clientID == "" || clientSecret == "" {
		return tokenError(c, http.StatusUnauthorized, "invalid_client", "client credentials required")
	}

	client, err := h.clientFinder.FindByClientID(ctx, clientID)
	if err != nil {
		c.Logger().Errorf("introspect: failed to find client: %v", err)
		return tokenError(c, http.StatusInternalServerError, "server_error", "")
	}
	if client == nil || client.Status != "active" {
		return tokenError(c, http.StatusUnauthorized, "invalid_client", "")
	}

	match, err := h.verifyPassword(clientSecret, client.ClientSecretHash)
	if err != nil || !match {
		return tokenError(c, http.StatusUnauthorized, "invalid_client", "")
	}

	token := c.FormValue("token")
	if token == "" {
		return c.JSON(http.StatusOK, map[string]interface{}{"active": false})
	}

	tokenTypeHint := c.FormValue("token_type_hint")

	// token_type_hint に基づいて検査（ヒントが間違っていても別の型を試す: RFC 7662 Section 2.1）
	switch tokenTypeHint {
	case "refresh_token":
		if resp := h.introspectRefreshToken(ctx, token); resp != nil {
			return c.JSON(http.StatusOK, resp)
		}
		if resp := h.introspectAccessToken(ctx, token); resp != nil {
			return c.JSON(http.StatusOK, resp)
		}
	default:
		// access_token またはヒントなし
		if resp := h.introspectAccessToken(ctx, token); resp != nil {
			return c.JSON(http.StatusOK, resp)
		}
		if resp := h.introspectRefreshToken(ctx, token); resp != nil {
			return c.JSON(http.StatusOK, resp)
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"active": false})
}

func (h *IntrospectHandler) introspectAccessToken(ctx context.Context, tokenString string) map[string]interface{} {
	result, err := h.tokenValidator.ValidateAccessToken(ctx, tokenString)
	if err != nil {
		return nil
	}

	// DB で失効チェック
	dbToken, err := h.accessTokenStore.FindByJTI(ctx, result.JTI)
	if err != nil || dbToken == nil || dbToken.RevokedAt != nil {
		return nil
	}

	return map[string]interface{}{
		"active":     true,
		"scope":      result.Scope,
		"client_id":  result.ClientID,
		"sub":        result.Subject.String(),
		"exp":        dbToken.ExpiresAt.Unix(),
		"token_type": "Bearer",
	}
}

func (h *IntrospectHandler) introspectRefreshToken(ctx context.Context, tokenString string) map[string]interface{} {
	tokenHash := h.sha256Hex(tokenString)
	rt, err := h.refreshTokenStore.FindByTokenHash(ctx, tokenHash)
	if err != nil || rt == nil {
		return nil
	}

	if rt.RevokedAt != nil || rt.ExpiresAt.Before(time.Now()) {
		return nil
	}

	if rt.AbsoluteExpiresAt != nil && rt.AbsoluteExpiresAt.Before(time.Now()) {
		return nil
	}

	return map[string]interface{}{
		"active":     true,
		"exp":        rt.ExpiresAt.Unix(),
		"token_type": "refresh_token",
	}
}
