package oidc

import (
	"github.com/labstack/echo/v4"
)

// SecurityHeadersMiddleware は全レスポンスにセキュリティヘッダーを付与する。
func SecurityHeadersMiddleware(isSecure bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()

			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

			if isSecure {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}

			// デフォルト Cache-Control（個別エンドポイントで上書きされる場合はそちらが優先）
			if h.Get("Cache-Control") == "" {
				h.Set("Cache-Control", "no-store")
			}

			return next(c)
		}
	}
}
