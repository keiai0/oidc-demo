package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/audit"
)

// MFATOTPVerifyHandler は POST /internal/mfa/totp/verify を処理する。
// ログインフローでの MFA 検証用エンドポイント。
type MFATOTPVerifyHandler struct {
	mfaSvc  *MFATOTPService
	authSvc *AuthService
	audit   *audit.AuditLogger
	logger  *slog.Logger
}

// NewMFATOTPVerifyHandler は MFATOTPVerifyHandler を生成する。
func NewMFATOTPVerifyHandler(mfaSvc *MFATOTPService, authSvc *AuthService, auditLog *audit.AuditLogger, logger *slog.Logger) *MFATOTPVerifyHandler {
	return &MFATOTPVerifyHandler{mfaSvc: mfaSvc, authSvc: authSvc, audit: auditLog, logger: logger}
}

type verifyRequest struct {
	Code            string `json:"code"`
	RedirectAfterMFA string `json:"redirect_after_mfa"`
}

// Handle はログインフロー MFA 検証リクエストを処理する。
// TOTP 検証成功後、セッションの pending_mfa を false にし、AMR/ACR を更新する。
func (h *MFATOTPVerifyHandler) Handle(c echo.Context) error {
	ctx := c.Request().Context()

	// セッション検証（pending_mfa=true のセッションも有効）
	session, err := validateSessionFromCookie(c, h.authSvc)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req verifyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
	}
	if req.Code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "code is required"})
	}

	// TOTP 検証 → セッション更新
	if err := h.mfaSvc.VerifyLogin(ctx, session, req.Code); err != nil {
		if errors.Is(err, ErrInvalidTOTPCode) {
			h.audit.LogEvent(ctx, audit.EventMFAFailure,
				audit.UserAttr(session.UserID.String()), audit.IPAttr(c.RealIP()), audit.MethodAttr("totp"), audit.ResultAttr("invalid_code"),
			)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_code", "error_description": "TOTP code is invalid"})
		}
		if errors.Is(err, ErrMFANotPending) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "mfa_not_pending", "error_description": "session is not pending MFA verification"})
		}
		if errors.Is(err, ErrMFANotConfigured) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "mfa_not_configured"})
		}
		h.logger.ErrorContext(ctx, "TOTP verify error", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	h.audit.LogEvent(ctx, audit.EventMFASuccess,
		audit.UserAttr(session.UserID.String()), audit.IPAttr(c.RealIP()), audit.MethodAttr("totp"), audit.ResultAttr("success"),
	)

	resp := map[string]interface{}{"success": true}
	if req.RedirectAfterMFA != "" {
		resp["redirect_to"] = req.RedirectAfterMFA
	}

	return c.JSON(http.StatusOK, resp)
}
