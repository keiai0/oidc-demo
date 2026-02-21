package management

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// NewAuthMiddleware は op_admin_session Cookie を検証する Echo ミドルウェアを返す。
func NewAuthMiddleware(adminAuthSvc *AdminAuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie(adminCookieName)
			if err == nil {
				sessionID, parseErr := uuid.Parse(cookie.Value)
				if parseErr == nil {
					if _, validateErr := adminAuthSvc.ValidateSession(c.Request().Context(), sessionID); validateErr == nil {
						return next(c)
					}
				}
			}

			return c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:            "unauthorized",
				ErrorDescription: "invalid or missing credentials",
			})
		}
	}
}
