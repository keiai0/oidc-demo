package auth

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

// MFATOTPDisableHandler は DELETE /internal/mfa/totp を処理する。
type MFATOTPDisableHandler struct {
	mfaTOTPSvc          *MFATOTPService
	authSvc             *AuthService
	userWithCredsFinder UserFinderWithCredentials
	verifyPassword      PasswordVerifyFunc
}

// NewMFATOTPDisableHandler は MFATOTPDisableHandler を生成する。
func NewMFATOTPDisableHandler(
	mfaTOTPSvc *MFATOTPService,
	authSvc *AuthService,
	userWithCredsFinder UserFinderWithCredentials,
	verifyPassword PasswordVerifyFunc,
) *MFATOTPDisableHandler {
	return &MFATOTPDisableHandler{
		mfaTOTPSvc:          mfaTOTPSvc,
		authSvc:             authSvc,
		userWithCredsFinder: userWithCredsFinder,
		verifyPassword:      verifyPassword,
	}
}

type mfaTOTPDisableRequest struct {
	Password string `json:"password"`
}

// Handle は DELETE /internal/mfa/totp を処理する。
// パスワード再確認の上で TOTP MFA を無効化する。
func (h *MFATOTPDisableHandler) Handle(c echo.Context) error {
	ctx := c.Request().Context()

	session, err := validateSessionFromCookie(c, h.authSvc)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	if !session.IsFullyAuthenticated() {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "mfa_required"})
	}

	var req mfaTOTPDisableRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
	}
	if req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "password is required"})
	}

	// パスワード再確認
	user, err := h.userWithCredsFinder.FindByIDWithCredentials(ctx, session.UserID)
	if err != nil || user == nil {
		c.Logger().Errorf("TOTP disable: find user error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	passwordHash := findPasswordHash(user.Credentials)
	if passwordHash == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
	}

	match, err := h.verifyPassword(req.Password, passwordHash)
	if err != nil {
		c.Logger().Errorf("TOTP disable: verify password error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	if !match {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
	}

	if err := h.mfaTOTPSvc.Disable(ctx, session.UserID); err != nil {
		if errors.Is(err, ErrMFANotConfigured) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "not_found", "error_description": "TOTP is not configured"})
		}
		c.Logger().Errorf("TOTP disable error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "totp_disabled"})
}
