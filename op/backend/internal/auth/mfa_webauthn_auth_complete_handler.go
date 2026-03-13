package auth

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

// MFAWebAuthnAuthCompleteHandler は POST /internal/mfa/webauthn/authenticate/complete を処理する。
type MFAWebAuthnAuthCompleteHandler struct {
	mfaSvc  *MFAWebAuthnService
	authSvc *AuthService
}

// NewMFAWebAuthnAuthCompleteHandler は MFAWebAuthnAuthCompleteHandler を生成する。
func NewMFAWebAuthnAuthCompleteHandler(mfaSvc *MFAWebAuthnService, authSvc *AuthService) *MFAWebAuthnAuthCompleteHandler {
	return &MFAWebAuthnAuthCompleteHandler{mfaSvc: mfaSvc, authSvc: authSvc}
}

// Handle は WebAuthn 認証完了リクエストを処理する。
// リクエストボディは go-webauthn ライブラリが直接パースするため、c.Bind() は使用しない。
// redirect_after_mfa はクエリパラメータで受け取る。
func (h *MFAWebAuthnAuthCompleteHandler) Handle(c echo.Context) error {
	ctx := c.Request().Context()

	session, err := validateSessionFromCookie(c, h.authSvc)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	if err := h.mfaSvc.CompleteAuthentication(ctx, session, c.Request()); err != nil {
		if errors.Is(err, ErrWebAuthnCloneDetected) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "clone_detected", "error_description": "authenticator clone detected"})
		}
		if errors.Is(err, ErrMFANotPending) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "mfa_not_pending"})
		}
		if errors.Is(err, ErrMFANotConfigured) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "mfa_not_configured"})
		}
		c.Logger().Errorf("WebAuthn auth complete error: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "authentication_failed", "error_description": err.Error()})
	}

	resp := map[string]interface{}{"success": true}
	if redirectTo := c.QueryParam("redirect_after_mfa"); redirectTo != "" {
		resp["redirect_to"] = redirectTo
	}

	return c.JSON(http.StatusOK, resp)
}
