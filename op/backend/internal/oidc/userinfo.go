package oidc

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type UserInfoHandler struct {
	tokenValidator   TokenValidator
	userFinder       UserFinder
	accessTokenStore AccessTokenStore
}

func NewUserInfoHandler(
	tokenValidator TokenValidator,
	userFinder UserFinder,
	accessTokenStore AccessTokenStore,
) *UserInfoHandler {
	return &UserInfoHandler{
		tokenValidator:   tokenValidator,
		userFinder:       userFinder,
		accessTokenStore: accessTokenStore,
	}
}

// Handle は GET /{tenant_code}/userinfo を処理する
// 仕様参照: OIDC Core 1.0 Section 5.3
func (h *UserInfoHandler) Handle(c echo.Context) error {
	// Bearer トークン取得
	authHeader := c.Request().Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		c.Response().Header().Set("WWW-Authenticate", `Bearer`)
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// アクセストークン検証 (JWT 署名検証 + 有効期限)
	result, err := h.tokenValidator.ValidateAccessToken(c.Request().Context(), tokenString)
	if err != nil {
		c.Response().Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
	}

	// DB でアクセストークンの失効チェック
	dbToken, err := h.accessTokenStore.FindByJTI(c.Request().Context(), result.JTI)
	if err != nil || dbToken == nil || dbToken.RevokedAt != nil {
		c.Response().Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
	}

	// ユーザー情報取得
	user, err := h.userFinder.FindByID(c.Request().Context(), result.Subject)
	if err != nil || user == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	// スコープに応じたクレーム構築
	claims := map[string]interface{}{
		"sub": user.ID.String(),
	}

	scopes := strings.Split(result.Scope, " ")

	if containsScope(scopes, "profile") {
		if user.Name != nil {
			claims["name"] = *user.Name
		}
		claims["updated_at"] = user.UpdatedAt.Unix()
	}

	if containsScope(scopes, "email") {
		claims["email"] = user.Email
		claims["email_verified"] = user.EmailVerified
	}

	return c.JSON(http.StatusOK, claims)
}
