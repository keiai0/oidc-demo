package oidc

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

type AuthorizeHandler struct {
	tenantFinder           TenantFinder
	clientFinder           ClientFinder
	tenantClientChecker    TenantClientChecker
	authCodeStore          AuthorizationCodeStore
	consentStore           ConsentStore
	sessionValidator       SessionValidator
	parStore               PARStore
	authDetailTypeFinder   AuthorizationDetailTypeFinder
	loginPageURL           string
	demoMode               bool
}

func NewAuthorizeHandler(
	tenantFinder TenantFinder,
	clientFinder ClientFinder,
	tenantClientChecker TenantClientChecker,
	authCodeStore AuthorizationCodeStore,
	consentStore ConsentStore,
	sessionValidator SessionValidator,
	parStore PARStore,
	authDetailTypeFinder AuthorizationDetailTypeFinder,
	loginPageURL string,
	demoMode bool,
) *AuthorizeHandler {
	return &AuthorizeHandler{
		tenantFinder:         tenantFinder,
		clientFinder:         clientFinder,
		tenantClientChecker:  tenantClientChecker,
		authCodeStore:        authCodeStore,
		consentStore:         consentStore,
		sessionValidator:     sessionValidator,
		parStore:             parStore,
		authDetailTypeFinder: authDetailTypeFinder,
		loginPageURL:         loginPageURL,
		demoMode:             demoMode,
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

	// PAR (RFC 9126): request_uri パラメータの処理
	requestURI := c.QueryParam("request_uri")
	var responseType, clientID, redirectURI, scope, state, nonce, codeChallenge, codeChallengeMethod, prompt, maxAgeStr, acrValuesStr, claimsParam, authorizationDetailsParam string

	if requestURI != "" {
		// request_uri が指定されている場合、他のパラメータは client_id 以外禁止 (RFC 9126 Section 4)
		clientID = c.QueryParam("client_id")

		if h.parStore == nil {
			return errorResponseDirect(c, "invalid_request", "PAR is not supported")
		}

		par, err := h.parStore.FindByRequestURI(ctx, requestURI)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
		}
		if par == nil {
			return errorResponseDirect(c, "invalid_request", "unknown request_uri")
		}
		if par.IsUsed() {
			return errorResponseDirect(c, "invalid_request", "request_uri already used")
		}
		if par.IsExpired() {
			return errorResponseDirect(c, "invalid_request", "request_uri expired")
		}

		// 使用済みにマーク
		if err := h.parStore.MarkAsUsed(ctx, par.ID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
		}

		// PAR に保存されたパラメータを復元
		var params map[string]string
		if err := json.Unmarshal(par.Parameters, &params); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
		}

		responseType = params["response_type"]
		if clientID == "" {
			clientID = params["client_id"]
		}
		redirectURI = params["redirect_uri"]
		scope = params["scope"]
		state = params["state"]
		nonce = params["nonce"]
		codeChallenge = params["code_challenge"]
		codeChallengeMethod = params["code_challenge_method"]
		prompt = params["prompt"]
		maxAgeStr = params["max_age"]
		acrValuesStr = params["acr_values"]
		claimsParam = params["claims"]
		authorizationDetailsParam = params["authorization_details"]

		// PAR 解決後: リクエスト URL を通常パラメータに書き換える。
		// ログイン後のリダイレクト先（redirect_after_login）が request_uri ではなく
		// 解決済みパラメータを含むようにし、二重使用エラーを防ぐ。
		resolvedQuery := url.Values{}
		for k, v := range params {
			if v != "" {
				resolvedQuery.Set(k, v)
			}
		}
		c.Request().URL.RawQuery = resolvedQuery.Encode()
	} else {
		// 通常のリクエストパラメータ取得
		responseType = c.QueryParam("response_type")
		clientID = c.QueryParam("client_id")
		redirectURI = c.QueryParam("redirect_uri")
		scope = c.QueryParam("scope")
		state = c.QueryParam("state")
		nonce = c.QueryParam("nonce")
		codeChallenge = c.QueryParam("code_challenge")
		codeChallengeMethod = c.QueryParam("code_challenge_method")
		prompt = c.QueryParam("prompt")
		maxAgeStr = c.QueryParam("max_age")
		acrValuesStr = c.QueryParam("acr_values")
		claimsParam = c.QueryParam("claims")
		authorizationDetailsParam = c.QueryParam("authorization_details")
	}

	// prompt パラメータ解析・検証 (OIDC Core 1.0 Section 3.1.2.1)
	prompts := parsePrompt(prompt)
	if hasPrompt(prompts, "none") && len(prompts) > 1 {
		// "none" は他の値と組み合わせ不可
		return errorResponseDirect(c, "invalid_request", "prompt=none cannot be combined with other values")
	}

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

	// テナント-クライアント紐づきチェック（中間テーブル参照）
	belongs, err := h.tenantClientChecker.ExistsByTenantAndClient(ctx, tenant.ID, client.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	if !belongs {
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

	// PKCE 検証（デモモード時はスキップ）
	if client.RequirePKCE && !h.demoMode {
		if codeChallenge == "" {
			return errorRedirect(c, redirectURI, state, "invalid_request", "code_challenge is required")
		}
		if codeChallengeMethod != "S256" {
			return errorRedirect(c, redirectURI, state, "invalid_request", "only S256 code_challenge_method is supported")
		}
	}

	// セッション確認
	var session *model.Session
	if cookie, err := c.Cookie("op_session"); err == nil {
		if sid, err := uuid.Parse(cookie.Value); err == nil {
			s, err := h.sessionValidator.ValidateSession(ctx, sid)
			if err == nil && s != nil {
				// テナントが一致するか確認
				if s.TenantID == tenant.ID {
					session = s
				}
			}
		}
	}

	// PendingMFA チェック: MFA 検証待ちのセッションを処理
	if session != nil && session.PendingMFA {
		if time.Since(session.AuthTime) > 5*time.Minute {
			// MFA タイムアウト → ログインからやり直し
			session = nil
		} else if session.MfaSetupRequired {
			// MFA 未設定 + テナント強制 → セットアップページへ
			return h.redirectToMFASetup(c, tenantCode)
		} else {
			// MFA 設定済み → 検証ページへ
			return h.redirectToMFA(c, tenantCode)
		}
	}

	// prompt=login → 再認証を要求
	if hasPrompt(prompts, "login") {
		session = nil
	}

	// max_age チェック (OIDC Core 1.0 Section 3.1.2.1)
	if maxAgeStr != "" && session != nil {
		maxAge, err := strconv.Atoi(maxAgeStr)
		if err == nil && maxAge >= 0 {
			if time.Since(session.AuthTime) > time.Duration(maxAge)*time.Second {
				session = nil // 認証経過時間が max_age を超えている → 再認証
			}
		}
	}

	// acr_values チェック
	if acrValuesStr != "" && session != nil {
		requestedACRs := strings.Split(acrValuesStr, " ")
		acrSatisfied := false
		for _, acr := range requestedACRs {
			if session.ACR == acr {
				acrSatisfied = true
				break
			}
		}
		if !acrSatisfied {
			session = nil // 要求 ACR を満たさない → 再認証
		}
	}

	// prompt=none のチェック
	if hasPrompt(prompts, "none") && session == nil {
		return errorRedirect(c, redirectURI, state, "login_required", "")
	}

	// セッションがなければログインページにリダイレクト
	if session == nil {
		return h.redirectToLogin(c, tenantCode)
	}

	// 同意チェック (OIDC Core 1.0 Section 3.1.2.4)
	consentRequired := false
	if hasPrompt(prompts, "consent") {
		// prompt=consent → 同意済みでも常に同意画面を表示
		consentRequired = true
	} else {
		consent, err := h.consentStore.FindByUserAndClient(ctx, session.UserID, client.ID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
		}
		if consent == nil || !consent.CoversScopes(scopes) {
			consentRequired = true
		}
	}

	if consentRequired {
		if hasPrompt(prompts, "none") {
			return errorRedirect(c, redirectURI, state, "consent_required", "")
		}
		return h.redirectToConsent(c, tenantCode, client.ID.String(), client.Name, scope, authorizationDetailsParam)
	}

	// claims パラメータの検証 (OIDC Core 1.0 Section 5.5)
	if claimsParam != "" {
		if _, err := ParseClaimsRequest(claimsParam); err != nil {
			return errorRedirect(c, redirectURI, state, "invalid_request", "invalid claims parameter")
		}
	}

	// authorization_details パラメータの検証 (RFC 9396 Section 2)
	if authorizationDetailsParam != "" {
		parsedDetails, err := ParseAuthorizationDetails(authorizationDetailsParam)
		if err != nil {
			return errorRedirect(c, redirectURI, state, "invalid_request", "invalid authorization_details: "+err.Error())
		}
		if err := ValidateAuthorizationDetails(ctx, parsedDetails, tenant.ID, h.authDetailTypeFinder); err != nil {
			return errorRedirect(c, redirectURI, state, "invalid_request", err.Error())
		}
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
	var claimsParamPtr *string
	if claimsParam != "" {
		claimsParamPtr = &claimsParam
	}
	var authDetailsPtr *string
	if authorizationDetailsParam != "" {
		authDetailsPtr = &authorizationDetailsParam
	}

	authCode := &model.AuthorizationCode{
		SessionID:           session.ID,
		ClientID:            client.ID,
		Code:                code,
		RedirectURI:         redirectURI,
		Scope:               scope,
		Nonce:               noncePtr,
		CodeChallenge:        challengePtr,
		CodeChallengeMethod: methodPtr,
		ClaimsParam:         claimsParamPtr,
		AuthorizationDetails: authDetailsPtr,
		ExpiresAt:            time.Now().Add(time.Duration(tenant.AuthCodeLifetime) * time.Second),
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

// redirectToMFASetup は MFA セットアップページにリダイレクトする。
// テナントが MFA を強制しているがユーザーが未設定の場合に使用する。
func (h *AuthorizeHandler) redirectToMFASetup(c echo.Context, tenantCode string) error {
	setupURL, err := url.Parse(h.loginPageURL + "/mfa/setup")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	q := setupURL.Query()
	q.Set("tenant_code", tenantCode)
	q.Set("redirect_after_mfa", c.Request().URL.String())
	setupURL.RawQuery = q.Encode()

	return c.Redirect(http.StatusFound, setupURL.String())
}

// redirectToMFA は MFA 検証ページにリダイレクトする。
// ログインリダイレクトと同じパターンで、authorize URL 全体を保存する。
func (h *AuthorizeHandler) redirectToMFA(c echo.Context, tenantCode string) error {
	mfaURL, err := url.Parse(h.loginPageURL + "/mfa/verify")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	q := mfaURL.Query()
	q.Set("tenant_code", tenantCode)
	q.Set("redirect_after_mfa", c.Request().URL.String())
	mfaURL.RawQuery = q.Encode()

	return c.Redirect(http.StatusFound, mfaURL.String())
}

// redirectToConsent は同意画面にリダイレクトする。
// ログインリダイレクトと同じパターンで、authorize URL 全体を保存する。
func (h *AuthorizeHandler) redirectToConsent(c echo.Context, tenantCode, clientID, clientName, scope, authorizationDetails string) error {
	consentURL, err := url.Parse(h.loginPageURL + "/consent")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	q := consentURL.Query()
	q.Set("tenant_code", tenantCode)
	q.Set("client_id", clientID)
	q.Set("client_name", clientName)
	q.Set("scope", scope)
	q.Set("redirect_after_consent", c.Request().URL.String())
	if authorizationDetails != "" {
		q.Set("authorization_details", authorizationDetails)
	}
	consentURL.RawQuery = q.Encode()

	return c.Redirect(http.StatusFound, consentURL.String())
}

// parsePrompt はスペース区切りの prompt パラメータを解析する。
func parsePrompt(prompt string) []string {
	if prompt == "" {
		return nil
	}
	return strings.Split(prompt, " ")
}

// hasPrompt は指定した prompt 値が含まれているかを返す。
func hasPrompt(prompts []string, target string) bool {
	for _, p := range prompts {
		if p == target {
			return true
		}
	}
	return false
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
