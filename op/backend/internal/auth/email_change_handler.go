package auth

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/labstack/echo/v4"
)

var emailRegexp = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// EmailChangeHandler はメールアドレス変更リクエスト・確認を処理する。
type EmailChangeHandler struct {
	emailChangeSvc *EmailChangeService
	authSvc        *AuthService
}

// NewEmailChangeHandler は EmailChangeHandler を生成する。
func NewEmailChangeHandler(emailChangeSvc *EmailChangeService, authSvc *AuthService) *EmailChangeHandler {
	return &EmailChangeHandler{
		emailChangeSvc: emailChangeSvc,
		authSvc:        authSvc,
	}
}

type emailChangeRequest struct {
	NewEmail string `json:"new_email"`
}

type emailVerifyRequest struct {
	Token string `json:"token"`
}

// HandleRequest は POST /internal/email/change-request を処理する。
// 認証済みユーザーが新しいメールアドレスへの確認メール送信を要求する。
func (h *EmailChangeHandler) HandleRequest(c echo.Context) error {
	ctx := c.Request().Context()

	session, err := validateSessionFromCookie(c, h.authSvc)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	// MFA 待ちセッションは不可（完全に認証済みである必要がある）
	if !session.IsFullyAuthenticated() {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "mfa_required"})
	}

	var req emailChangeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
	}
	if req.NewEmail == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "new_email is required"})
	}
	if !emailRegexp.MatchString(req.NewEmail) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "new_email format is invalid"})
	}

	if err := h.emailChangeSvc.RequestChange(ctx, session.UserID, req.NewEmail); err != nil {
		c.Logger().Errorf("email change request error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	// ユーザー列挙防止のため常に成功レスポンスを返す
	return c.JSON(http.StatusOK, map[string]string{"status": "verification_sent"})
}

// HandleVerify は POST /internal/email/verify を処理する。
// メールで受け取ったトークンを検証し、メールアドレスを更新する。
// このエンドポイントはセッション不要（メールリンク経由のフロー）。
func (h *EmailChangeHandler) HandleVerify(c echo.Context) error {
	ctx := c.Request().Context()

	var req emailVerifyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
	}
	if req.Token == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "token is required"})
	}

	if err := h.emailChangeSvc.VerifyChange(ctx, req.Token); err != nil {
		if errors.Is(err, ErrEmailChangeTokenInvalid) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "token_invalid"})
		}
		c.Logger().Errorf("email change verify error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "email_changed"})
}
