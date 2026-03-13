package auth

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// MFAWebAuthnRegisterCompleteHandler は POST /internal/mfa/webauthn/register/complete を処理する。
type MFAWebAuthnRegisterCompleteHandler struct {
	mfaSvc  *MFAWebAuthnService
	authSvc *AuthService
}

// NewMFAWebAuthnRegisterCompleteHandler は MFAWebAuthnRegisterCompleteHandler を生成する。
func NewMFAWebAuthnRegisterCompleteHandler(mfaSvc *MFAWebAuthnService, authSvc *AuthService) *MFAWebAuthnRegisterCompleteHandler {
	return &MFAWebAuthnRegisterCompleteHandler{mfaSvc: mfaSvc, authSvc: authSvc}
}

// Handle は WebAuthn 登録完了リクエストを処理する。
// リクエストボディは go-webauthn ライブラリが直接パースするため、c.Bind() は使用しない。
// クレデンシャル名はクエリパラメータ name で受け取る。
func (h *MFAWebAuthnRegisterCompleteHandler) Handle(c echo.Context) error {
	ctx := c.Request().Context()

	session, err := validateSessionFromCookie(c, h.authSvc)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	credName := c.QueryParam("name")

	if err := h.mfaSvc.CompleteRegistration(ctx, session.UserID, session, c.Request(), credName); err != nil {
		c.Logger().Errorf("WebAuthn register complete error: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "registration_failed", "error_description": err.Error()})
	}

	resp := map[string]interface{}{"success": true}

	return c.JSON(http.StatusOK, resp)
}
