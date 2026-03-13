package auth

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// MFAWebAuthnRegisterBeginHandler は POST /internal/mfa/webauthn/register/begin を処理する。
type MFAWebAuthnRegisterBeginHandler struct {
	mfaSvc     *MFAWebAuthnService
	authSvc    *AuthService
	userFinder UserFinder
}

// NewMFAWebAuthnRegisterBeginHandler は MFAWebAuthnRegisterBeginHandler を生成する。
func NewMFAWebAuthnRegisterBeginHandler(mfaSvc *MFAWebAuthnService, authSvc *AuthService, userFinder UserFinder) *MFAWebAuthnRegisterBeginHandler {
	return &MFAWebAuthnRegisterBeginHandler{mfaSvc: mfaSvc, authSvc: authSvc, userFinder: userFinder}
}

type registerBeginRequest struct {
	Name string `json:"name"`
}

// Handle は WebAuthn 登録開始リクエストを処理する。
// セッション Cookie 必須。CredentialCreationOptions を返す。
func (h *MFAWebAuthnRegisterBeginHandler) Handle(c echo.Context) error {
	ctx := c.Request().Context()

	session, err := validateSessionFromCookie(c, h.authSvc)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req registerBeginRequest
	_ = c.Bind(&req) // name は任意

	// ユーザー情報取得
	user, err := h.userFinder.FindByID(ctx, session.UserID)
	if err != nil || user == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	userName := user.Email
	if userName == "" {
		userName = user.LoginID
	}
	displayName := userName
	if user.Name != nil && *user.Name != "" {
		displayName = *user.Name
	}

	creationJSON, err := h.mfaSvc.BeginRegistration(ctx, session.UserID, userName, displayName, session)
	if err != nil {
		c.Logger().Errorf("WebAuthn register begin error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	return c.JSONBlob(http.StatusOK, creationJSON)
}
