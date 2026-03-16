package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/audit"
)

// MFAWebAuthnAuthCompleteHandler は POST /internal/mfa/webauthn/authenticate/complete を処理する。
type MFAWebAuthnAuthCompleteHandler struct {
	mfaSvc  *MFAWebAuthnService
	authSvc *AuthService
	audit   *audit.AuditLogger
	logger  *slog.Logger
}

// NewMFAWebAuthnAuthCompleteHandler は MFAWebAuthnAuthCompleteHandler を生成する。
func NewMFAWebAuthnAuthCompleteHandler(mfaSvc *MFAWebAuthnService, authSvc *AuthService, auditLog *audit.AuditLogger, logger *slog.Logger) *MFAWebAuthnAuthCompleteHandler {
	return &MFAWebAuthnAuthCompleteHandler{mfaSvc: mfaSvc, authSvc: authSvc, audit: auditLog, logger: logger}
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
			h.audit.LogEvent(ctx, audit.EventMFAFailure,
				audit.UserAttr(session.UserID.String()), audit.IPAttr(c.RealIP()), audit.MethodAttr("webauthn"), audit.ResultAttr("clone_detected"),
			)
			return c.JSON(http.StatusForbidden, map[string]string{"error": "clone_detected", "error_description": "authenticator clone detected"})
		}
		if errors.Is(err, ErrMFANotPending) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "mfa_not_pending"})
		}
		if errors.Is(err, ErrMFANotConfigured) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "mfa_not_configured"})
		}
		h.audit.LogEvent(ctx, audit.EventMFAFailure,
			audit.UserAttr(session.UserID.String()), audit.IPAttr(c.RealIP()), audit.MethodAttr("webauthn"), audit.ResultAttr("failed"),
		)
		h.logger.ErrorContext(ctx, "WebAuthn auth complete error", "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "authentication_failed", "error_description": err.Error()})
	}

	h.audit.LogEvent(ctx, audit.EventMFASuccess,
		audit.UserAttr(session.UserID.String()), audit.IPAttr(c.RealIP()), audit.MethodAttr("webauthn"), audit.ResultAttr("success"),
	)

	resp := map[string]interface{}{"success": true}
	if redirectTo := c.QueryParam("redirect_after_mfa"); redirectTo != "" {
		resp["redirect_to"] = redirectTo
	}

	return c.JSON(http.StatusOK, resp)
}
