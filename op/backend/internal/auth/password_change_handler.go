package auth

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// PasswordChangeHandler は POST /internal/password/change を処理する。
type PasswordChangeHandler struct {
	passwordSvc *PasswordService
	authSvc     *AuthService
	isSecure    bool
}

// NewPasswordChangeHandler は PasswordChangeHandler を生成する。
func NewPasswordChangeHandler(passwordSvc *PasswordService, authSvc *AuthService, isSecure bool) *PasswordChangeHandler {
	return &PasswordChangeHandler{
		passwordSvc: passwordSvc,
		authSvc:     authSvc,
		isSecure:    isSecure,
	}
}

type passwordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// Handle は POST /internal/password/change を処理する。
// セッション認証済みユーザーのパスワードを変更する。
func (h *PasswordChangeHandler) Handle(c echo.Context) error {
	// セッション検証
	cookie, err := c.Cookie("op_session")
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	sessionID, err := uuid.Parse(cookie.Value)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	session, err := h.authSvc.ValidateSession(c.Request().Context(), sessionID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req passwordChangeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "current_password and new_password are required"})
	}

	if len(req.NewPassword) < 8 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "new_password must be at least 8 characters"})
	}

	err = h.passwordSvc.ChangePassword(c.Request().Context(), session.UserID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid_credentials", "error_description": "current password is incorrect"})
		}
		if errors.Is(err, ErrPasswordSameAsCurrent) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "password_same_as_current", "error_description": "new password must differ from current password"})
		}
		if errors.Is(err, ErrPasswordInHistory) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "password_in_history", "error_description": "this password was recently used"})
		}
		c.Logger().Errorf("password change error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "password_changed"})
}
