package auth

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

// PasswordResetRequestHandler は POST /internal/password/reset-request を処理する。
type PasswordResetRequestHandler struct {
	resetSvc *PasswordResetService
}

// NewPasswordResetRequestHandler は PasswordResetRequestHandler を生成する。
func NewPasswordResetRequestHandler(resetSvc *PasswordResetService) *PasswordResetRequestHandler {
	return &PasswordResetRequestHandler{resetSvc: resetSvc}
}

type resetRequestBody struct {
	TenantCode string `json:"tenant_code"`
	Email      string `json:"email"`
}

// Handle は POST /internal/password/reset-request を処理する。
// ユーザー列挙防止のため、常に 200 OK を返す。
func (h *PasswordResetRequestHandler) Handle(c echo.Context) error {
	var req resetRequestBody
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
	}

	if req.TenantCode == "" || req.Email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "tenant_code and email are required"})
	}

	if err := h.resetSvc.RequestReset(c.Request().Context(), req.TenantCode, req.Email); err != nil {
		c.Logger().Errorf("password reset request error: %v", err)
		// ユーザー列挙防止のため、エラーでも 200 を返す
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// PasswordResetHandler は POST /internal/password/reset を処理する。
type PasswordResetHandler struct {
	resetSvc *PasswordResetService
}

// NewPasswordResetHandler は PasswordResetHandler を生成する。
func NewPasswordResetHandler(resetSvc *PasswordResetService) *PasswordResetHandler {
	return &PasswordResetHandler{resetSvc: resetSvc}
}

type resetBody struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// Handle は POST /internal/password/reset を処理する。
func (h *PasswordResetHandler) Handle(c echo.Context) error {
	var req resetBody
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
	}

	if req.Token == "" || req.NewPassword == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "token and new_password are required"})
	}

	if len(req.NewPassword) < 8 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "new_password must be at least 8 characters"})
	}

	err := h.resetSvc.ExecuteReset(c.Request().Context(), req.Token, req.NewPassword)
	if err != nil {
		if errors.Is(err, ErrResetTokenInvalid) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_token", "error_description": "reset token is invalid or expired"})
		}
		if errors.Is(err, ErrPasswordSameAsCurrent) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "password_same_as_current", "error_description": "new password must differ from current password"})
		}
		if errors.Is(err, ErrPasswordInHistory) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "password_in_history", "error_description": "this password was recently used"})
		}
		c.Logger().Errorf("password reset error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "password_reset"})
}
