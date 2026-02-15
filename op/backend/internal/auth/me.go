package auth

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type MeHandler struct {
	authSvc    *AuthService
	userFinder UserFinder
}

func NewMeHandler(authSvc *AuthService, userFinder UserFinder) *MeHandler {
	return &MeHandler{authSvc: authSvc, userFinder: userFinder}
}

// Handle は GET /internal/me を処理する。
// op_session クッキーからセッションを検証し、ユーザー情報を返す。
func (h *MeHandler) Handle(c echo.Context) error {
	cookie, err := c.Cookie("op_session")
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "no_session"})
	}

	sessionID, err := uuid.Parse(cookie.Value)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid_session"})
	}

	session, err := h.authSvc.ValidateSession(c.Request().Context(), sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) || errors.Is(err, ErrSessionExpired) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "session_expired"})
		}
		c.Logger().Errorf("session validation error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	user, err := h.userFinder.FindByID(c.Request().Context(), session.UserID)
	if err != nil || user == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"session_id": session.ID.String(),
		"tenant_id":  session.TenantID.String(),
		"user": map[string]interface{}{
			"id":    user.ID.String(),
			"name":  user.Name,
			"email": user.Email,
		},
	})
}
