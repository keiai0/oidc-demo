package auth

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// MFAWebAuthnCredentialsHandler は WebAuthn クレデンシャルの一覧・削除を処理する。
type MFAWebAuthnCredentialsHandler struct {
	mfaSvc  *MFAWebAuthnService
	authSvc *AuthService
}

// NewMFAWebAuthnCredentialsHandler は MFAWebAuthnCredentialsHandler を生成する。
func NewMFAWebAuthnCredentialsHandler(mfaSvc *MFAWebAuthnService, authSvc *AuthService) *MFAWebAuthnCredentialsHandler {
	return &MFAWebAuthnCredentialsHandler{mfaSvc: mfaSvc, authSvc: authSvc}
}

type credentialResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// HandleList は GET /internal/mfa/webauthn/credentials を処理する。
func (h *MFAWebAuthnCredentialsHandler) HandleList(c echo.Context) error {
	ctx := c.Request().Context()

	session, err := validateSessionFromCookie(c, h.authSvc)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	creds, err := h.mfaSvc.ListCredentials(ctx, session.UserID)
	if err != nil {
		c.Logger().Errorf("WebAuthn list credentials error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	resp := make([]credentialResponse, len(creds))
	for i, c := range creds {
		resp[i] = credentialResponse{
			ID:        c.ID.String(),
			Name:      c.Name,
			CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return c.JSON(http.StatusOK, resp)
}

// HandleDelete は DELETE /internal/mfa/webauthn/credentials/:id を処理する。
func (h *MFAWebAuthnCredentialsHandler) HandleDelete(c echo.Context) error {
	ctx := c.Request().Context()

	session, err := validateSessionFromCookie(c, h.authSvc)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	credID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "invalid credential ID"})
	}

	if err := h.mfaSvc.DeleteCredential(ctx, session.UserID, credID); err != nil {
		if errors.Is(err, ErrMFANotConfigured) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "not_found"})
		}
		if errors.Is(err, ErrWebAuthnCredentialNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "not_found"})
		}
		c.Logger().Errorf("WebAuthn delete credential error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"success": true})
}
