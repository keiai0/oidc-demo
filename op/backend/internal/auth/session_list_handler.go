package auth

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// SessionListHandler はユーザー向けのセッション一覧・失効を処理する。
type SessionListHandler struct {
	sessionStore   SessionStore
	sessionLister  SessionLister
	sessionRevoker SessionRevoker
	authSvc        *AuthService
}

// NewSessionListHandler は SessionListHandler を生成する。
func NewSessionListHandler(
	sessionStore SessionStore,
	sessionLister SessionLister,
	sessionRevoker SessionRevoker,
	authSvc *AuthService,
) *SessionListHandler {
	return &SessionListHandler{
		sessionStore:   sessionStore,
		sessionLister:  sessionLister,
		sessionRevoker: sessionRevoker,
		authSvc:        authSvc,
	}
}

type sessionListResponse struct {
	ID        string `json:"id"`
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
	IsCurrent bool   `json:"is_current"`
}

// HandleList は GET /internal/sessions を処理する。
// ユーザーのアクティブセッション一覧を返す。
func (h *SessionListHandler) HandleList(c echo.Context) error {
	ctx := c.Request().Context()

	session, err := validateSessionFromCookie(c, h.authSvc)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	if !session.IsFullyAuthenticated() {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "mfa_required"})
	}

	sessions, err := h.sessionLister.FindActiveByUserID(ctx, session.UserID)
	if err != nil {
		c.Logger().Errorf("session list error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	resp := make([]sessionListResponse, len(sessions))
	for i, s := range sessions {
		resp[i] = sessionListResponse{
			ID:        s.ID.String(),
			IPAddress: s.IPAddress,
			UserAgent: s.UserAgent,
			CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			ExpiresAt: s.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
			IsCurrent: s.ID == session.ID,
		}
	}

	return c.JSON(http.StatusOK, resp)
}

// HandleRevoke は DELETE /internal/sessions/:id を処理する。
// 指定されたセッションを失効させる。現在のセッションは失効不可。
func (h *SessionListHandler) HandleRevoke(c echo.Context) error {
	ctx := c.Request().Context()

	session, err := validateSessionFromCookie(c, h.authSvc)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	if !session.IsFullyAuthenticated() {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "mfa_required"})
	}

	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "invalid session ID"})
	}

	// 現在のセッションは失効不可
	if targetID == session.ID {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "cannot_revoke_current_session"})
	}

	// 所有権確認: 対象セッションがこのユーザーのものであることを確認
	target, err := h.sessionStore.FindByID(ctx, targetID)
	if err != nil {
		c.Logger().Errorf("session revoke: find session error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	if target == nil || target.UserID != session.UserID {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not_found"})
	}

	if err := h.sessionRevoker.Revoke(ctx, targetID); err != nil {
		c.Logger().Errorf("session revoke error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "revoked"})
}
