package scim

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// NewAuthMiddleware は SCIM エンドポイント用の Bearer トークン認証ミドルウェアを返す。
// アクセストークンの scope に "scim" が含まれていることを検証する。
func NewAuthMiddleware(tokenValidator TokenValidator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Content-Type を SCIM 用に設定
			c.Response().Header().Set("Content-Type", "application/scim+json")

			// Authorization: Bearer <token> を抽出
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return scimError(c, http.StatusUnauthorized, "Authorization header required", "")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return scimError(c, http.StatusUnauthorized, "Bearer token required", "")
			}

			tokenStr := parts[1]

			// アクセストークン検証
			result, err := tokenValidator.ValidateAccessToken(c.Request().Context(), tokenStr)
			if err != nil {
				return scimError(c, http.StatusUnauthorized, "Invalid access token", "")
			}

			// scope に "scim" が含まれているか確認
			scopes := strings.Split(result.Scope, " ")
			hasScim := false
			for _, s := range scopes {
				if s == "scim" {
					hasScim = true
					break
				}
			}
			if !hasScim {
				return scimError(c, http.StatusForbidden, "Insufficient scope: scim required", "")
			}

			return next(c)
		}
	}
}
