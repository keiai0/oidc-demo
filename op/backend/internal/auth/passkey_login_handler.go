package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

// PasskeyLoginHandler はパスキーによるパスワードレスログインのエンドポイントを処理する。
type PasskeyLoginHandler struct {
	svc      *PasskeyLoginService
	isSecure bool
}

// NewPasskeyLoginHandler は PasskeyLoginHandler を生成する。
func NewPasskeyLoginHandler(svc *PasskeyLoginService, isSecure bool) *PasskeyLoginHandler {
	return &PasskeyLoginHandler{svc: svc, isSecure: isSecure}
}

// HandleBegin は POST /internal/passkey/login/begin を処理する。
// 未ログイン状態で呼ばれる。Discoverable Login のチャレンジを発行する。
func (h *PasskeyLoginHandler) HandleBegin(c echo.Context) error {
	ctx := c.Request().Context()

	optionsJSON, challengeID, err := h.svc.BeginPasskeyLogin(ctx)
	if err != nil {
		c.Logger().Errorf("passkey login begin error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	// challenge_id をレスポンスに含めて返す（フロントが complete 時に送り返す）
	resp := map[string]interface{}{
		"challenge_id": challengeID,
	}

	// options を resp にマージ
	var options map[string]interface{}
	if err := echoJSONUnmarshal(optionsJSON, &options); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	for k, v := range options {
		resp[k] = v
	}

	return c.JSON(http.StatusOK, resp)
}

type passkeyLoginCompleteRequest struct {
	TenantCode  string `json:"tenant_code"`
	ChallengeID string `json:"challenge_id"`
}

// HandleComplete は POST /internal/passkey/login/complete を処理する。
// フロントから Assertion レスポンスと challenge_id を受け取り、検証してセッションを確立する。
// 注: go-webauthn は http.Request.Body を直接読むため、Assertion データは別途渡す必要がある。
// → tenant_code と challenge_id はクエリパラメータで受け取り、Body は go-webauthn に渡す。
func (h *PasskeyLoginHandler) HandleComplete(c echo.Context) error {
	ctx := c.Request().Context()

	challengeID := c.QueryParam("challenge_id")
	tenantCode := c.QueryParam("tenant_code")

	if challengeID == "" || tenantCode == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "challenge_id and tenant_code are required"})
	}

	output, err := h.svc.CompletePasskeyLogin(ctx, challengeID, tenantCode, c.Request(), c.RealIP(), c.Request().UserAgent())
	if err != nil {
		if errors.Is(err, ErrWebAuthnCloneDetected) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "clone_detected"})
		}
		if errors.Is(err, ErrInvalidCredentials) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
		}
		if errors.Is(err, ErrAccountLocked) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "account_locked"})
		}
		c.Logger().Errorf("passkey login complete error: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "authentication_failed", "error_description": err.Error()})
	}

	// セッション Cookie 設定
	cookie := &http.Cookie{
		Name:     "op_session",
		Value:    output.SessionID.String(),
		Path:     "/",
		HttpOnly: true,
		Secure:   h.isSecure,
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"session_id": output.SessionID.String(),
		"user": map[string]interface{}{
			"id":    output.User.ID.String(),
			"name":  output.User.Name,
			"email": output.User.Email,
		},
	})
}

// echoJSONUnmarshal は json.RawMessage を map にデコードするヘルパー。
func echoJSONUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
