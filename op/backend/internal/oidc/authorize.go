package oidc

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

type AuthorizeHandler struct {
	tenantFinder     TenantFinder
	clientFinder     ClientFinder
	authCodeStore    AuthorizationCodeStore
	sessionValidator SessionValidator
	loginPageURL     string
}

func NewAuthorizeHandler(
	tenantFinder TenantFinder,
	clientFinder ClientFinder,
	authCodeStore AuthorizationCodeStore,
	sessionValidator SessionValidator,
	loginPageURL string,
) *AuthorizeHandler {
	return &AuthorizeHandler{
		tenantFinder:     tenantFinder,
		clientFinder:     clientFinder,
		authCodeStore:    authCodeStore,
		sessionValidator: sessionValidator,
		loginPageURL:     loginPageURL,
	}
}

// Handle は GET /{tenant_code}/authorize を処理する
// 仕様参照: RFC 6749 Section 4.1.1, OIDC Core 1.0 Section 3.1.2.1
func (h *AuthorizeHandler) Handle(c echo.Context) error {
	ctx := c.Request().Context()
	tenantCode := c.Param("tenant_code")

	// テナント検証
	tenant, err := h.tenantFinder.FindByCode(ctx, tenantCode)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	if tenant == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not_found"})
	}

	// リクエストパラメータ取得
	responseType := c.QueryParam("response_type")
	clientID := c.QueryParam("client_id")
	redirectURI := c.QueryParam("redirect_uri")
	scope := c.QueryParam("scope")
	state := c.QueryParam("state")
	nonce := c.QueryParam("nonce")
	codeChallenge := c.QueryParam("code_challenge")
	codeChallengeMethod := c.QueryParam("code_challenge_method")
	prompt := c.QueryParam("prompt")

	// response_type 検証 (MUST: "code" のみ)
	if responseType != "code" {
		return errorResponseDirect(c, "unsupported_response_type", "only response_type=code is supported")
	}

	// client_id 検証
	if clientID == "" {
		return errorResponseDirect(c, "invalid_request", "client_id is required")
	}

	client, err := h.clientFinder.FindByClientIDWithRedirectURIs(ctx, clientID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	if client == nil || client.Status != "active" {
		return errorResponseDirect(c, "invalid_request", "unknown client_id")
	}

	// テナント一致チェック
	if client.TenantID != tenant.ID {
		return errorResponseDirect(c, "invalid_request", "client does not belong to this tenant")
	}

	// redirect_uri 完全一致検証 (MUST: RFC 6749 Section 3.1.2.3)
	// 検証失敗時はリダイレクトしない（MUST: RFC 6749 Section 4.1.2.1）
	if redirectURI == "" {
		return errorResponseDirect(c, "invalid_request", "redirect_uri is required")
	}
	if !isRegisteredRedirectURI(client.RedirectURIs, redirectURI) {
		return errorResponseDirect(c, "invalid_request", "redirect_uri mismatch")
	}

	// ここから先はエラーをredirect_uriにリダイレクトで返す

	// scope 検証 ("openid" 必須)
	scopes := strings.Split(scope, " ")
	if !containsScope(scopes, "openid") {
		return errorRedirect(c, redirectURI, state, "invalid_scope", "openid scope is required")
	}

	// grant_type サポート確認
	if !client.HasGrantType("authorization_code") {
		return errorRedirect(c, redirectURI, state, "unauthorized_client", "client does not support authorization_code grant")
	}

	// PKCE 検証
	if client.RequirePKCE {
		if codeChallenge == "" {
			return errorRedirect(c, redirectURI, state, "invalid_request", "code_challenge is required")
		}
		if codeChallengeMethod != "S256" {
			return errorRedirect(c, redirectURI, state, "invalid_request", "only S256 code_challenge_method is supported")
		}
	}

	// セッション確認
	var sessionID *uuid.UUID
	if cookie, err := c.Cookie("op_session"); err == nil {
		if sid, err := uuid.Parse(cookie.Value); err == nil {
			session, err := h.sessionValidator.ValidateSession(ctx, sid)
			if err == nil && session != nil {
				// テナントが一致するか確認
				if session.TenantID == tenant.ID {
					sessionID = &session.ID
				}
			}
		}
	}

	// prompt パラメータ処理
	if prompt == "none" && sessionID == nil {
		return errorRedirect(c, redirectURI, state, "login_required", "")
	}

	if prompt == "login" {
		sessionID = nil // 再認証を要求
	}

	// セッションがなければログインページにリダイレクト
	if sessionID == nil {
		return h.redirectToLogin(c, tenantCode)
	}

	// 認可コード発行
	codeBytes := make([]byte, 32)
	if _, err := rand.Read(codeBytes); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	code := hex.EncodeToString(codeBytes)

	var noncePtr *string
	if nonce != "" {
		noncePtr = &nonce
	}
	var challengePtr *string
	if codeChallenge != "" {
		challengePtr = &codeChallenge
	}
	var methodPtr *string
	if codeChallengeMethod != "" {
		methodPtr = &codeChallengeMethod
	}

	authCode := &model.AuthorizationCode{
		SessionID:           *sessionID,
		ClientID:            client.ID,
		Code:                code,
		RedirectURI:         redirectURI,
		Scope:               scope,
		Nonce:               noncePtr,
		CodeChallenge:       challengePtr,
		CodeChallengeMethod: methodPtr,
		ExpiresAt:           time.Now().Add(time.Duration(tenant.AuthCodeLifetime) * time.Second),
	}

	if err := h.authCodeStore.Create(ctx, authCode); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	// redirect_uri に認可コードとstateを付与してリダイレクト
	redirectURL, err := url.Parse(redirectURI)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	q := redirectURL.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	redirectURL.RawQuery = q.Encode()

	return c.Redirect(http.StatusFound, redirectURL.String())
}

// redirectToLogin はログインページにリダイレクトする。
// 現在のauthorize URLをredirect_after_loginパラメータに含める。
func (h *AuthorizeHandler) redirectToLogin(c echo.Context, tenantCode string) error {
	loginURL, err := url.Parse(h.loginPageURL + "/login")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	q := loginURL.Query()
	q.Set("tenant_code", tenantCode)
	// 認可リクエスト全体を redirect_after_login に保存
	q.Set("redirect_after_login", c.Request().URL.String())
	loginURL.RawQuery = q.Encode()

	return c.Redirect(http.StatusFound, loginURL.String())
}

func isRegisteredRedirectURI(registeredURIs []model.RedirectURI, uri string) bool {
	for _, r := range registeredURIs {
		if r.URI == uri {
			return true
		}
	}
	return false
}

func containsScope(scopes []string, target string) bool {
	for _, s := range scopes {
		if s == target {
			return true
		}
	}
	return false
}

// errorResponseDirect は redirect_uri にリダイレクトせずに直接エラーレスポンスを返す。
// redirect_uri/client_id 検証失敗時に使用（RFC 6749 Section 4.1.2.1）。
func errorResponseDirect(c echo.Context, errCode, errDescription string) error {
	body := map[string]string{"error": errCode}
	if errDescription != "" {
		body["error_description"] = errDescription
	}
	return c.JSON(http.StatusBadRequest, body)
}

// errorRedirect は redirect_uri にエラーをクエリパラメータとしてリダイレクトする。
func errorRedirect(c echo.Context, redirectURI, state, errCode, errDescription string) error {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
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

	return c.Redirect(http.StatusFound, u.String())
}
