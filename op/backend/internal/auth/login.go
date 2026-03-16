package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/audit"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/metrics"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

type LoginHandler struct {
	authSvc  *AuthService
	audit    *audit.AuditLogger
	logger   *slog.Logger
	isSecure bool
}

func NewLoginHandler(authSvc *AuthService, auditLog *audit.AuditLogger, logger *slog.Logger, isSecure bool) *LoginHandler {
	return &LoginHandler{authSvc: authSvc, audit: auditLog, logger: logger, isSecure: isSecure}
}

type loginRequest struct {
	TenantCode string `json:"tenant_code" validate:"required"`
	LoginID    string `json:"login_id" validate:"required"`
	Password   string `json:"password" validate:"required"`
}

// Handle は POST /internal/login を処理する
func (h *LoginHandler) Handle(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
	}

	if req.TenantCode == "" || req.LoginID == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "tenant_code, login_id, and password are required"})
	}

	input := &model.LoginInput{
		TenantCode: req.TenantCode,
		LoginID:    req.LoginID,
		Password:   req.Password,
		IPAddress:  c.RealIP(),
		UserAgent:  c.Request().UserAgent(),
	}

	// セッション固定攻撃対策: 既存セッション Cookie があれば旧セッション ID を渡す
	if oldCookie, err := c.Cookie("op_session"); err == nil && oldCookie.Value != "" {
		if oldID, parseErr := uuid.Parse(oldCookie.Value); parseErr == nil {
			input.OldSessionID = &oldID
		}
	}

	ctx := c.Request().Context()
	output, err := h.authSvc.Login(ctx, input)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			h.audit.LogEvent(ctx, audit.EventLoginFailure,
				audit.IPAttr(c.RealIP()), audit.TenantAttr(req.TenantCode), audit.ResultAttr("invalid_credentials"),
			)
			metrics.LoginTotal.WithLabelValues("failure").Inc()
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
		}
		if errors.Is(err, ErrAccountLocked) {
			h.audit.LogEvent(ctx, audit.EventLoginLocked,
				audit.IPAttr(c.RealIP()), audit.TenantAttr(req.TenantCode), audit.ResultAttr("account_locked"),
			)
			metrics.LoginTotal.WithLabelValues("locked").Inc()
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "account_locked", "error_description": "account is temporarily locked due to too many failed login attempts"})
		}
		h.logger.ErrorContext(ctx, "login error", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	h.audit.LogEvent(ctx, audit.EventLoginSuccess,
		audit.UserAttr(output.User.ID.String()), audit.IPAttr(c.RealIP()), audit.TenantAttr(req.TenantCode), audit.ResultAttr("success"),
	)
	metrics.LoginTotal.WithLabelValues("success").Inc()
	metrics.ActiveSessions.Inc()

	// セッションクッキーを設定
	cookie := &http.Cookie{
		Name:     "op_session",
		Value:    output.SessionID.String(),
		Path:     "/",
		HttpOnly: true,
		Secure:   h.isSecure,
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)

	resp := map[string]interface{}{
		"session_id": output.SessionID.String(),
		"user": map[string]interface{}{
			"id":    output.User.ID.String(),
			"name":  output.User.Name,
			"email": output.User.Email,
		},
	}
	if output.MFARequired {
		resp["mfa_required"] = true
		resp["mfa_methods"] = output.MFAMethods
	}
	if output.MFASetupRequired {
		resp["mfa_setup_required"] = true
	}
	resp["passkey_registered"] = output.PasskeyRegistered

	return c.JSON(http.StatusOK, resp)
}
