package auth

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

// MFABackupCodeHandler は MFA バックアップコードの生成・検証を処理する。
type MFABackupCodeHandler struct {
	backupCodeSvc *BackupCodeService
	authSvc       *AuthService
}

// NewMFABackupCodeHandler は MFABackupCodeHandler を生成する。
func NewMFABackupCodeHandler(backupCodeSvc *BackupCodeService, authSvc *AuthService) *MFABackupCodeHandler {
	return &MFABackupCodeHandler{
		backupCodeSvc: backupCodeSvc,
		authSvc:       authSvc,
	}
}

type backupCodeVerifyRequest struct {
	Code string `json:"code"`
}

// HandleGenerate は POST /internal/mfa/backup-codes/generate を処理する。
// 新しいバックアップコードを10個生成し、平文をレスポンスに含める（1回限り）。
func (h *MFABackupCodeHandler) HandleGenerate(c echo.Context) error {
	ctx := c.Request().Context()

	session, err := validateSessionFromCookie(c, h.authSvc)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	if !session.IsFullyAuthenticated() {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "mfa_required"})
	}

	codes, err := h.backupCodeSvc.Generate(ctx, session.UserID)
	if err != nil {
		c.Logger().Errorf("backup code generate error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"codes": codes})
}

// HandleVerify は POST /internal/mfa/backup-codes/verify を処理する。
// バックアップコードを検証し、MFA 認証を完了する。
// pending_mfa=true のセッションからも使用可能（MFA 手段を失った場合の最終手段）。
func (h *MFABackupCodeHandler) HandleVerify(c echo.Context) error {
	ctx := c.Request().Context()

	// IsValid() のみ確認（pending_mfa=true でも使用可能）
	session, err := validateSessionFromCookie(c, h.authSvc)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req backupCodeVerifyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
	}
	if req.Code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "code is required"})
	}

	if err := h.backupCodeSvc.Verify(ctx, session, req.Code); err != nil {
		if errors.Is(err, ErrBackupCodeInvalid) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid_backup_code"})
		}
		c.Logger().Errorf("backup code verify error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "mfa_completed"})
}
