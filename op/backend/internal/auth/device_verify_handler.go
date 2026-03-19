package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/audit"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// DeviceAuthorizationStore はデバイス認可リクエストの検索・更新操作を定義する。
type DeviceAuthorizationStore interface {
	// FindByUserCode は user_code でデバイス認可リクエストを検索する（クライアント情報をプリロード）。
	FindByUserCode(ctx context.Context, userCode string) (*model.DeviceAuthorizationRequest, error)
	// UpdateStatus はリクエストのステータスを更新する。
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, sessionID *uuid.UUID) error
}

// DeviceVerifyHandler は /internal/device/* の Internal API を処理する。
// デバイスフローでユーザーが user_code を入力して承認/拒否するための API。
type DeviceVerifyHandler struct {
	deviceAuthStore DeviceAuthorizationStore
	authSvc         *AuthService
	audit           *audit.AuditLogger
}

// NewDeviceVerifyHandler は DeviceVerifyHandler を生成する。
func NewDeviceVerifyHandler(
	deviceAuthStore DeviceAuthorizationStore,
	authSvc *AuthService,
	auditLog *audit.AuditLogger,
) *DeviceVerifyHandler {
	return &DeviceVerifyHandler{
		deviceAuthStore: deviceAuthStore,
		authSvc:         authSvc,
		audit:           auditLog,
	}
}

type deviceVerifyResponse struct {
	UserCode   string `json:"user_code"`
	Scope      string `json:"scope"`
	ClientName string `json:"client_name"`
	ExpiresAt  string `json:"expires_at"`
}

// HandleVerify は GET /internal/device/verify?user_code=XXXX を処理する。
// user_code でデバイス認可リクエストを検索し、スコープとクライアント情報を返す。
func (h *DeviceVerifyHandler) HandleVerify(c echo.Context) error {
	// セッション検証
	_, err := h.validateSession(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	userCode := c.QueryParam("user_code")
	if userCode == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user_code is required"})
	}

	req, err := h.deviceAuthStore.FindByUserCode(c.Request().Context(), userCode)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	if req == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "invalid_user_code"})
	}

	if req.IsExpired() {
		return c.JSON(http.StatusGone, map[string]string{"error": "expired"})
	}

	if !req.IsPending() {
		return c.JSON(http.StatusConflict, map[string]string{"error": "already_processed"})
	}

	return c.JSON(http.StatusOK, deviceVerifyResponse{
		UserCode:   req.UserCode,
		Scope:      req.Scope,
		ClientName: req.Client.Name,
		ExpiresAt:  req.ExpiresAt.Format(time.RFC3339),
	})
}

type deviceDecisionRequest struct {
	UserCode string `json:"user_code"`
}

// HandleApprove は POST /internal/device/approve を処理する。
// 認証済みユーザーがデバイス認可リクエストを承認する。
func (h *DeviceVerifyHandler) HandleApprove(c echo.Context) error {
	session, err := h.validateSession(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var body deviceDecisionRequest
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
	}
	if body.UserCode == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user_code is required"})
	}

	ctx := c.Request().Context()
	req, err := h.deviceAuthStore.FindByUserCode(ctx, body.UserCode)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	if req == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "invalid_user_code"})
	}

	if req.IsExpired() {
		return c.JSON(http.StatusGone, map[string]string{"error": "expired"})
	}
	if !req.IsPending() {
		return c.JSON(http.StatusConflict, map[string]string{"error": "already_processed"})
	}

	// 承認: ステータスを approved に変更し、session_id をリンク
	sessionID := session.ID
	if err := h.deviceAuthStore.UpdateStatus(ctx, req.ID, "approved", &sessionID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	h.audit.LogEvent(ctx, audit.EventDeviceApproved,
		audit.UserAttr(session.UserID.String()),
		audit.ClientAttr(req.Client.ClientID),
	)

	return c.JSON(http.StatusOK, map[string]string{"status": "approved"})
}

// HandleDeny は POST /internal/device/deny を処理する。
// 認証済みユーザーがデバイス認可リクエストを拒否する。
func (h *DeviceVerifyHandler) HandleDeny(c echo.Context) error {
	session, err := h.validateSession(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var body deviceDecisionRequest
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
	}
	if body.UserCode == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "user_code is required"})
	}

	ctx := c.Request().Context()
	req, err := h.deviceAuthStore.FindByUserCode(ctx, body.UserCode)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	if req == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "invalid_user_code"})
	}

	if req.IsExpired() {
		return c.JSON(http.StatusGone, map[string]string{"error": "expired"})
	}
	if !req.IsPending() {
		return c.JSON(http.StatusConflict, map[string]string{"error": "already_processed"})
	}

	if err := h.deviceAuthStore.UpdateStatus(ctx, req.ID, "denied", nil); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	h.audit.LogEvent(ctx, audit.EventDeviceDenied,
		audit.UserAttr(session.UserID.String()),
		audit.ClientAttr(req.Client.ClientID),
	)

	return c.JSON(http.StatusOK, map[string]string{"status": "denied"})
}

// validateSession はセッション Cookie からセッションを検証する。
func (h *DeviceVerifyHandler) validateSession(c echo.Context) (*model.Session, error) {
	cookie, err := c.Cookie("op_session")
	if err != nil {
		return nil, err
	}
	sessionID, err := uuid.Parse(cookie.Value)
	if err != nil {
		return nil, err
	}
	return h.authSvc.ValidateSession(c.Request().Context(), sessionID)
}
