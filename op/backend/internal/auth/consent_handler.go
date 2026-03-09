package auth

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// ConsentStore は同意記録の永続化操作を定義する（auth パッケージ用）。
type ConsentStore interface {
	Upsert(ctx context.Context, consent *model.UserConsent) error
}

// ConsentHandler は POST /internal/consent を処理する。
type ConsentHandler struct {
	authSvc      *AuthService
	consentStore ConsentStore
}

// NewConsentHandler は ConsentHandler を生成する。
func NewConsentHandler(authSvc *AuthService, consentStore ConsentStore) *ConsentHandler {
	return &ConsentHandler{
		authSvc:      authSvc,
		consentStore: consentStore,
	}
}

type consentRequest struct {
	ClientID             string   `json:"client_id"`
	Scopes               []string `json:"scopes"`
	Approved             bool     `json:"approved"`
	RedirectAfterConsent string   `json:"redirect_after_consent"`
	RedirectURI          string   `json:"redirect_uri"`
	State                string   `json:"state"`
}

type consentResponse struct {
	RedirectTo string `json:"redirect_to"`
}

// Handle は POST /internal/consent を処理する。
func (h *ConsentHandler) Handle(c echo.Context) error {
	// セッション検証
	cookie, err := c.Cookie("op_session")
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	sessionID, err := uuid.Parse(cookie.Value)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	session, err := h.authSvc.ValidateSession(c.Request().Context(), sessionID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req consentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
	}

	if req.ClientID == "" || len(req.Scopes) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "client_id and scopes are required"})
	}

	if !req.Approved {
		// 拒否: redirect_uri に error=access_denied でリダイレクト
		if req.RedirectURI == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "redirect_uri is required for denial"})
		}
		redirectTo := buildErrorRedirect(req.RedirectURI, req.State, "access_denied", "user denied the consent request")
		return c.JSON(http.StatusOK, consentResponse{RedirectTo: redirectTo})
	}

	// 承認: 同意記録を upsert
	// client_id (外部ID) から内部 UUID を特定する必要があるが、
	// consent の永続化では UserID + ClientID(DB UUID) が必要。
	// ここでは authorize URL を redirect_after_consent に含めて返し、
	// フロントエンドがそこにリダイレクトすることで authorize が再度呼ばれ、
	// 同意チェックが通る設計にする。

	// client_id は外部 ID のため、UUID を解釈しない。
	// 代わりに scope をスペース区切りで保存する。
	clientUUID, err := uuid.Parse(req.ClientID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "invalid client_id"})
	}

	consent := &model.UserConsent{
		UserID:    session.UserID,
		ClientID:  clientUUID,
		Scopes:    strings.Join(req.Scopes, " "),
		GrantedAt: time.Now(),
	}
	if err := h.consentStore.Upsert(c.Request().Context(), consent); err != nil {
		c.Logger().Errorf("consent upsert error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	// authorize URL にリダイレクト（同意チェックが通るようになる）
	if req.RedirectAfterConsent == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "redirect_after_consent is required"})
	}

	return c.JSON(http.StatusOK, consentResponse{RedirectTo: req.RedirectAfterConsent})
}

func buildErrorRedirect(redirectURI, state, errCode, errDescription string) string {
	u, err := parseURL(redirectURI)
	if err != nil {
		return redirectURI
	}
	q := u.Query()
	q.Set("error", errCode)
	if errDescription != "" {
		q.Set("error_description", errDescription)
	}
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func parseURL(rawURL string) (*url.URL, error) {
	return url.Parse(rawURL) //nolint:wrapcheck
}
