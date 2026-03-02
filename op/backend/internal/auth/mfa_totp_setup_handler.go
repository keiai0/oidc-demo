package auth

import (
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// MFATOTPSetupHandler は POST /internal/mfa/totp/setup を処理する。
type MFATOTPSetupHandler struct {
	mfaSvc     *MFATOTPService
	authSvc    *AuthService
	userFinder UserFinder
}

// NewMFATOTPSetupHandler は MFATOTPSetupHandler を生成する。
func NewMFATOTPSetupHandler(mfaSvc *MFATOTPService, authSvc *AuthService, userFinder UserFinder) *MFATOTPSetupHandler {
	return &MFATOTPSetupHandler{mfaSvc: mfaSvc, authSvc: authSvc, userFinder: userFinder}
}

// Handle は TOTP セットアップリクエストを処理する。
// セッション Cookie 必須。QR コードとシークレットを返す。
func (h *MFATOTPSetupHandler) Handle(c echo.Context) error {
	ctx := c.Request().Context()

	// セッション検証
	session, err := validateSessionFromCookie(c, h.authSvc)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	// ユーザー情報取得（accountName 用）
	user, err := h.userFinder.FindByID(ctx, session.UserID)
	if err != nil || user == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	accountName := user.Email
	if accountName == "" {
		accountName = user.LoginID
	}

	// TOTP セットアップ実行
	result, err := h.mfaSvc.Setup(ctx, session.UserID, accountName)
	if err != nil {
		if errors.Is(err, ErrMFAAlreadyConfigured) {
			return c.JSON(http.StatusConflict, map[string]string{"error": "mfa_already_configured", "error_description": "TOTP is already configured and enabled"})
		}
		c.Logger().Errorf("TOTP setup error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"secret":      result.Secret,
		"qr_code_uri": result.QRCodeURI,
		"qr_code_png": base64.StdEncoding.EncodeToString(result.QRCodePNG),
	})
}

// validateSessionFromCookie は Cookie からセッションを検証する共通ヘルパー。
func validateSessionFromCookie(c echo.Context, authSvc *AuthService) (*model.Session, error) {
	cookie, err := c.Cookie("op_session")
	if err != nil {
		return nil, err
	}

	sid, err := uuid.Parse(cookie.Value)
	if err != nil {
		return nil, err
	}

	return authSvc.ValidateSession(c.Request().Context(), sid)
}
