package auth

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// SessionRevoker はセッションの失効操作を定義する。
type SessionRevoker interface {
	Revoke(ctx context.Context, id uuid.UUID) error
}

// TokenRevokerBySession はセッションに紐づくトークンの一括失効を定義する。
type TokenRevokerBySession interface {
	RevokeBySessionID(ctx context.Context, sessionID uuid.UUID) error
}

// InternalLogoutHandler は POST /internal/logout を処理する。
// OP Frontend がセッションを破棄する際に使用する。
type InternalLogoutHandler struct {
	sessionRevoker       SessionRevoker
	accessTokenRevoker   TokenRevokerBySession
	refreshTokenRevoker  TokenRevokerBySession
	isSecure             bool
}

// NewInternalLogoutHandler は InternalLogoutHandler を生成する。
func NewInternalLogoutHandler(
	sessionRevoker SessionRevoker,
	accessTokenRevoker TokenRevokerBySession,
	refreshTokenRevoker TokenRevokerBySession,
	isSecure bool,
) *InternalLogoutHandler {
	return &InternalLogoutHandler{
		sessionRevoker:      sessionRevoker,
		accessTokenRevoker:  accessTokenRevoker,
		refreshTokenRevoker: refreshTokenRevoker,
		isSecure:            isSecure,
	}
}

// Handle は POST /internal/logout を処理する。
// op_session Cookie からセッションを特定し、セッション+関連トークンを失効する。
func (h *InternalLogoutHandler) Handle(c echo.Context) error {
	ctx := c.Request().Context()

	cookie, err := c.Cookie("op_session")
	if err != nil {
		// Cookie がなくても成功扱い（冪等性）
		return c.JSON(http.StatusOK, map[string]string{"status": "logged_out"})
	}

	sessionID, err := uuid.Parse(cookie.Value)
	if err != nil {
		h.clearSessionCookie(c)
		return c.JSON(http.StatusOK, map[string]string{"status": "logged_out"})
	}

	// セッション失効（エラーは無視: 冪等性）
	_ = h.sessionRevoker.Revoke(ctx, sessionID)
	_ = h.accessTokenRevoker.RevokeBySessionID(ctx, sessionID)
	_ = h.refreshTokenRevoker.RevokeBySessionID(ctx, sessionID)

	h.clearSessionCookie(c)

	return c.JSON(http.StatusOK, map[string]string{"status": "logged_out"})
}

func (h *InternalLogoutHandler) clearSessionCookie(c echo.Context) {
	cookie := &http.Cookie{
		Name:     "op_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.isSecure,
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)
}
