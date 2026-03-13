package auth

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

// MFAWebAuthnAuthBeginHandler は POST /internal/mfa/webauthn/authenticate/begin を処理する。
type MFAWebAuthnAuthBeginHandler struct {
	mfaSvc  *MFAWebAuthnService
	authSvc *AuthService
}

// NewMFAWebAuthnAuthBeginHandler は MFAWebAuthnAuthBeginHandler を生成する。
func NewMFAWebAuthnAuthBeginHandler(mfaSvc *MFAWebAuthnService, authSvc *AuthService) *MFAWebAuthnAuthBeginHandler {
	return &MFAWebAuthnAuthBeginHandler{mfaSvc: mfaSvc, authSvc: authSvc}
}

// Handle は WebAuthn 認証開始リクエストを処理する。
func (h *MFAWebAuthnAuthBeginHandler) Handle(c echo.Context) error {
	ctx := c.Request().Context()

	session, err := validateSessionFromCookie(c, h.authSvc)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	assertionJSON, err := h.mfaSvc.BeginAuthentication(ctx, session)
	if err != nil {
		if errors.Is(err, ErrWebAuthnNoCredentials) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "no_credentials", "error_description": "no WebAuthn credentials registered"})
		}
		c.Logger().Errorf("WebAuthn auth begin error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	return c.JSONBlob(http.StatusOK, assertionJSON)
}
