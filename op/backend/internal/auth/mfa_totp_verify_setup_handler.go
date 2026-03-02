package auth

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

// MFATOTPVerifySetupHandler は POST /internal/mfa/totp/verify-setup を処理する。
type MFATOTPVerifySetupHandler struct {
	mfaSvc  *MFATOTPService
	authSvc *AuthService
}

// NewMFATOTPVerifySetupHandler は MFATOTPVerifySetupHandler を生成する。
func NewMFATOTPVerifySetupHandler(mfaSvc *MFATOTPService, authSvc *AuthService) *MFATOTPVerifySetupHandler {
	return &MFATOTPVerifySetupHandler{mfaSvc: mfaSvc, authSvc: authSvc}
}

type verifySetupRequest struct {
	Code            string `json:"code"`
	RedirectAfterMFA string `json:"redirect_after_mfa"`
}

// Handle はセットアップ時の TOTP 検証リクエストを処理する。
// 検証成功で MFA を有効化する。ログインフロー内の場合はセッションの MFA 状態も更新する。
func (h *MFATOTPVerifySetupHandler) Handle(c echo.Context) error {
	ctx := c.Request().Context()

	// セッション検証
	session, err := validateSessionFromCookie(c, h.authSvc)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req verifySetupRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
	}
	if req.Code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "code is required"})
	}

	// TOTP 検証 → MFA 有効化（+ ログインフロー内ならセッション更新）
	if err := h.mfaSvc.VerifySetup(ctx, session.UserID, req.Code, session); err != nil {
		if errors.Is(err, ErrInvalidTOTPCode) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_code", "error_description": "TOTP code is invalid"})
		}
		if errors.Is(err, ErrMFANotConfigured) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "mfa_not_configured", "error_description": "TOTP setup has not been started"})
		}
		if errors.Is(err, ErrMFAAlreadyConfigured) {
			return c.JSON(http.StatusConflict, map[string]string{"error": "mfa_already_configured"})
		}
		c.Logger().Errorf("TOTP verify-setup error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	resp := map[string]interface{}{"success": true}
	if req.RedirectAfterMFA != "" {
		resp["redirect_to"] = req.RedirectAfterMFA
	}

	return c.JSON(http.StatusOK, resp)
}
