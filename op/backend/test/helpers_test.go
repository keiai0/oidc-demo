package test

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// =============================================================================
// フィクスチャヘルパー
// =============================================================================

// createTenant はテスト用テナントを DB に作成する。
func createTenant(t *testing.T, code, name string) *model.Tenant {
	t.Helper()
	tenant := &model.Tenant{
		Code:                 code,
		Name:                 name,
		SessionLifetime:      3600,
		AuthCodeLifetime:     60,
		AccessTokenLifetime:  3600,
		RefreshTokenLifetime: 2592000,
		IDTokenLifetime:      3600,
		MfaRequired:          false,
	}
	if err := testDB.Create(tenant).Error; err != nil {
		t.Fatalf("createTenant failed: %v", err)
	}
	return tenant
}

// ClientOption はクライアント作成時のオプション。
type ClientOption func(*model.Client)

// WithClientID はクライアント ID を指定する。
func WithClientID(clientID string) ClientOption {
	return func(c *model.Client) { c.ClientID = clientID }
}

// WithGrantTypes は許可する grant_type を指定する。
func WithGrantTypes(types ...string) ClientOption {
	return func(c *model.Client) { c.GrantTypes = model.StringSlice(types) }
}

// WithRequirePKCE は PKCE 必須を設定する。
func WithRequirePKCE(v bool) ClientOption {
	return func(c *model.Client) { c.RequirePKCE = v }
}

// WithTokenEndpointAuthMethod はクライアント認証方式を指定する。
func WithTokenEndpointAuthMethod(method string) ClientOption {
	return func(c *model.Client) { c.TokenEndpointAuthMethod = method }
}

// createClient はテスト用クライアントを DB に作成する。
// キャッシュ済みの secret hash を使用し、固定の testClientSecret を返す。
func createClient(t *testing.T, opts ...ClientOption) *model.Client {
	t.Helper()
	client := &model.Client{
		ClientID:                fmt.Sprintf("test-client-%s", uuid.New().String()[:8]),
		ClientSecretHash:        cachedSecretHash,
		Name:                    "Test Client",
		GrantTypes:              model.StringSlice{"authorization_code", "refresh_token"},
		ResponseTypes:           model.StringSlice{"code"},
		TokenEndpointAuthMethod: "client_secret_basic",
		RequirePKCE:             true,
		Status:                  "active",
		SubjectType:             "public",
	}
	for _, opt := range opts {
		opt(client)
	}
	if err := testDB.Create(client).Error; err != nil {
		t.Fatalf("createClient failed: %v", err)
	}
	return client
}

// createRedirectURI はリダイレクト URI を作成する。
func createRedirectURI(t *testing.T, clientDBID uuid.UUID, uri string) *model.RedirectURI {
	t.Helper()
	redirectURI := &model.RedirectURI{
		ClientDBID: clientDBID,
		URI:        uri,
	}
	if err := testDB.Create(redirectURI).Error; err != nil {
		t.Fatalf("createRedirectURI failed: %v", err)
	}
	return redirectURI
}

// linkTenantClient はテナントとクライアントを関連付ける。
func linkTenantClient(t *testing.T, tenantID, clientID uuid.UUID) {
	t.Helper()
	tc := &model.TenantClient{
		TenantID: tenantID,
		ClientID: clientID,
		Enabled:  true,
	}
	if err := testDB.Create(tc).Error; err != nil {
		t.Fatalf("linkTenantClient failed: %v", err)
	}
}

// createUser はテスト用ユーザーを DB に作成する（パスワード付き）。
func createUser(t *testing.T, tenantID uuid.UUID, loginID, email string) *model.User {
	t.Helper()
	name := "Test User"
	user := &model.User{
		TenantID:      tenantID,
		LoginID:       loginID,
		Email:         email,
		EmailVerified: true,
		Name:          &name,
		Status:        "active",
	}
	if err := testDB.Create(user).Error; err != nil {
		t.Fatalf("createUser failed: %v", err)
	}

	// Credential + PasswordCredential を作成
	cred := &model.Credential{
		UserID: user.ID,
		Type:   "password",
	}
	if err := testDB.Create(cred).Error; err != nil {
		t.Fatalf("createCredential failed: %v", err)
	}

	pwCred := &model.PasswordCredential{
		CredentialID: cred.ID,
		PasswordHash: cachedPasswordHash,
		Algorithm:    "argon2id",
	}
	if err := testDB.Create(pwCred).Error; err != nil {
		t.Fatalf("createPasswordCredential failed: %v", err)
	}

	return user
}

// createSession はテスト用セッションを DB に作成する。
func createSession(t *testing.T, userID, tenantID uuid.UUID) *model.Session {
	t.Helper()
	session := &model.Session{
		UserID:    userID,
		TenantID:  tenantID,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
		AuthTime:  time.Now(),
		AMR:       model.StringSlice{"pwd"},
		ACR:       "urn:mace:incommon:iap:bronze",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := testDB.Create(session).Error; err != nil {
		t.Fatalf("createSession failed: %v", err)
	}
	return session
}

// createConsent はテスト用の同意レコードを作成する。
func createConsent(t *testing.T, userID, clientID uuid.UUID, scopes string) *model.UserConsent {
	t.Helper()
	consent := &model.UserConsent{
		UserID:    userID,
		ClientID:  clientID,
		Scopes:    scopes,
		GrantedAt: time.Now(),
	}
	if err := testDB.Create(consent).Error; err != nil {
		t.Fatalf("createConsent failed: %v", err)
	}
	return consent
}

// createPostLogoutRedirectURI は post_logout_redirect_uri を作成する。
func createPostLogoutRedirectURI(t *testing.T, clientDBID uuid.UUID, uri string) {
	t.Helper()
	plru := &model.PostLogoutRedirectURI{
		ClientDBID: clientDBID,
		URI:        uri,
	}
	if err := testDB.Create(plru).Error; err != nil {
		t.Fatalf("createPostLogoutRedirectURI failed: %v", err)
	}
}

// cleanupTables は全テーブルを TRUNCATE CASCADE でクリーンアップする。
// sign_keys はテスト基盤で必要なので除外する。
func cleanupTables(t *testing.T) {
	t.Helper()
	tables := []string{
		"user_consents",
		"password_credentials",
		"credentials",
		"id_tokens",
		"access_tokens",
		"refresh_tokens",
		"authorization_codes",
		"sessions",
		"pushed_authorization_requests",
		"device_authorization_requests",
		"dpop_jti_cache",
		"redirect_uris",
		"post_logout_redirect_uris",
		"tenant_clients",
		"users",
		"clients",
		"tenants",
	}
	for _, table := range tables {
		if err := testDB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)).Error; err != nil {
			t.Fatalf("cleanupTables failed for %s: %v", table, err)
		}
	}
}

// =============================================================================
// PKCE ヘルパー
// =============================================================================

// generatePKCE は code_verifier と code_challenge（S256）を生成する。
func generatePKCE() (codeVerifier, codeChallenge string) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	codeVerifier = base64.RawURLEncoding.EncodeToString(buf)
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge = base64.RawURLEncoding.EncodeToString(hash[:])
	return
}

// =============================================================================
// HTTP ヘルパー
// =============================================================================

// ReqOption は HTTP リクエストのオプション。
type ReqOption func(*http.Request)

// withSessionCookie はセッション Cookie を付与する。
func withSessionCookie(sessionID uuid.UUID) ReqOption {
	return func(r *http.Request) {
		r.AddCookie(&http.Cookie{
			Name:  "op_session",
			Value: sessionID.String(),
		})
	}
}

// withBasicAuth は HTTP Basic 認証を設定する。
func withBasicAuth(clientID, clientSecret string) ReqOption {
	return func(r *http.Request) {
		r.SetBasicAuth(clientID, clientSecret)
	}
}

// withBearerToken は Bearer トークンを設定する。
func withBearerToken(token string) ReqOption {
	return func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+token)
	}
}

// doGET は GET リクエストを送信する。
func doGET(t *testing.T, path string, opts ...ReqOption) *http.Response {
	t.Helper()
	reqURL := serverURL + path
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	for _, opt := range opts {
		opt(req)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	return resp
}

// doPOSTForm は POST リクエスト（application/x-www-form-urlencoded）を送信する。
func doPOSTForm(t *testing.T, path string, form url.Values, opts ...ReqOption) *http.Response {
	t.Helper()
	reqURL := serverURL + path
	req, err := http.NewRequest(http.MethodPost, reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, opt := range opts {
		opt(req)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	return resp
}

// =============================================================================
// アサーション / パースヘルパー
// =============================================================================

// parseJSON はレスポンスボディを JSON パースする。
func parseJSON(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("JSON Unmarshal failed: %v\nbody: %s", err, string(body))
	}
	return result
}

// assertStatus はレスポンスのステータスコードを検証する。
func assertStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d; body: %s", expected, resp.StatusCode, string(body))
	}
}

// extractRedirectParam は Location ヘッダからクエリパラメータを取得する。
func extractRedirectParam(t *testing.T, resp *http.Response, param string) string {
	t.Helper()
	loc := resp.Header.Get("Location")
	if loc == "" {
		t.Fatal("Location header is empty")
	}
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("url.Parse(%q) failed: %v", loc, err)
	}
	return u.Query().Get(param)
}

// decodeJWTClaims は JWT のペイロード部をデコードする（署名検証なし）。
func decodeJWTClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("invalid JWT: expected 3 parts, got %d", len(parts))
	}
	// base64url デコード（パディング補完）
	payload := parts[1]
	if m := len(payload) % 4; m != 0 {
		payload += strings.Repeat("=", 4-m)
	}
	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(decoded, &claims); err != nil {
		t.Fatalf("JSON unmarshal claims failed: %v", err)
	}
	return claims
}

// =============================================================================
// フロー実行ヘルパー
// =============================================================================

// FlowResult は Authorization Code Flow の結果を保持する。
type FlowResult struct {
	AccessToken   string
	IDToken       string
	RefreshToken  string
	IDTokenClaims map[string]any
	Scope         string
	TokenResponse map[string]any
}

// FlowOption は Authorization Code Flow 実行時のオプション。
type FlowOption func(*flowConfig)

type flowConfig struct {
	scope string
	nonce string
}

// WithScope はスコープを指定する。
func WithScope(scope string) FlowOption {
	return func(fc *flowConfig) { fc.scope = scope }
}

// WithNonce は nonce を指定する。
func WithNonce(nonce string) FlowOption {
	return func(fc *flowConfig) { fc.nonce = nonce }
}

// setupFlowFixtures はフロー実行に必要なフィクスチャを作成する。
type flowFixtures struct {
	tenant  *model.Tenant
	client  *model.Client
	user    *model.User
	session *model.Session
}

func setupFlowFixtures(t *testing.T) *flowFixtures {
	t.Helper()
	tenant := createTenant(t, "test", "Test Tenant")
	client := createClient(t)
	createRedirectURI(t, client.ID, "http://localhost:3001/callback")
	linkTenantClient(t, tenant.ID, client.ID)
	user := createUser(t, tenant.ID, "testuser", "testuser@example.com")
	session := createSession(t, user.ID, tenant.ID)
	createConsent(t, user.ID, client.ID, "openid profile email offline_access")
	return &flowFixtures{tenant: tenant, client: client, user: user, session: session}
}

// performAuthCodeFlow は Authorization Code Flow を一気通貫で実行する。
func performAuthCodeFlow(t *testing.T, fixtures *flowFixtures, opts ...FlowOption) *FlowResult {
	t.Helper()

	cfg := &flowConfig{
		scope: "openid profile email",
		nonce: uuid.New().String(),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	codeVerifier, codeChallenge := generatePKCE()
	state := uuid.New().String()

	// 1. Authorize リクエスト
	authPath := fmt.Sprintf("/%s/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s&nonce=%s&code_challenge=%s&code_challenge_method=S256",
		fixtures.tenant.Code,
		url.QueryEscape(fixtures.client.ClientID),
		url.QueryEscape("http://localhost:3001/callback"),
		url.QueryEscape(cfg.scope),
		url.QueryEscape(state),
		url.QueryEscape(cfg.nonce),
		url.QueryEscape(codeChallenge),
	)

	resp := doGET(t, authPath, withSessionCookie(fixtures.session.ID))
	if resp.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("authorize: expected 302, got %d; body: %s", resp.StatusCode, string(body))
	}
	resp.Body.Close()

	code := extractRedirectParam(t, resp, "code")
	if code == "" {
		t.Fatalf("authorize: code not found in redirect; Location: %s", resp.Header.Get("Location"))
	}

	// state の検証
	returnedState := extractRedirectParam(t, resp, "state")
	if returnedState != state {
		t.Fatalf("state mismatch: got %q, want %q", returnedState, state)
	}

	// 2. Token リクエスト
	tokenForm := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {"http://localhost:3001/callback"},
		"code_verifier": {codeVerifier},
	}

	tokenResp := doPOSTForm(t, fmt.Sprintf("/%s/token", fixtures.tenant.Code), tokenForm,
		withBasicAuth(fixtures.client.ClientID, testClientSecret))

	if tokenResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(tokenResp.Body)
		tokenResp.Body.Close()
		t.Fatalf("token: expected 200, got %d; body: %s", tokenResp.StatusCode, string(body))
	}

	tokenBody := parseJSON(t, tokenResp)

	accessToken, _ := tokenBody["access_token"].(string)
	idToken, _ := tokenBody["id_token"].(string)
	refreshToken, _ := tokenBody["refresh_token"].(string)
	scopeReturned, _ := tokenBody["scope"].(string)

	var idTokenClaims map[string]any
	if idToken != "" {
		idTokenClaims = decodeJWTClaims(t, idToken)
	}

	return &FlowResult{
		AccessToken:   accessToken,
		IDToken:       idToken,
		RefreshToken:  refreshToken,
		IDTokenClaims: idTokenClaims,
		Scope:         scopeReturned,
		TokenResponse: tokenBody,
	}
}
