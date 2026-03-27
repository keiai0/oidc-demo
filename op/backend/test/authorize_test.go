package test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// Test_Authorize は Authorization Endpoint を検証する。
// OpenID Foundation 認定の「Basic OP」プロファイルに相当。
func Test_Authorize(t *testing.T) {
	cleanupTables(t)

	tenant := createTenant(t, "authz", "Authorize Tenant")
	client := createClient(t)
	createRedirectURI(t, client.ID, "http://localhost:3001/callback")
	linkTenantClient(t, tenant.ID, client.ID)
	user := createUser(t, tenant.ID, "authzuser", "authzuser@example.com")
	session := createSession(t, user.ID, tenant.ID)

	redirectURI := "http://localhost:3001/callback"

	buildAuthURL := func(overrides map[string]string) string {
		params := map[string]string{
			"response_type":         "code",
			"client_id":             client.ClientID,
			"redirect_uri":          redirectURI,
			"scope":                 "openid",
			"state":                 "test-state",
			"nonce":                 "test-nonce",
			"code_challenge":        "",
			"code_challenge_method": "S256",
		}
		// PKCE のデフォルト
		if params["code_challenge"] == "" {
			_, cc := generatePKCE()
			params["code_challenge"] = cc
		}
		for k, v := range overrides {
			if v == "" {
				delete(params, k)
			} else {
				params[k] = v
			}
		}
		q := url.Values{}
		for k, v := range params {
			q.Set(k, v)
		}
		return fmt.Sprintf("/%s/authorize?%s", tenant.Code, q.Encode())
	}

	t.Run("正常系: session + consent + PKCE -> 302 with code", func(t *testing.T) {
		createConsent(t, user.ID, client.ID, "openid profile email")
		defer testDB.Exec("DELETE FROM user_consents WHERE user_id = ?", user.ID)

		cv, cc := generatePKCE()
		_ = cv // code_verifier はトークン交換時に使う
		authURL := buildAuthURL(map[string]string{
			"code_challenge": cc,
		})

		resp := doGET(t, authURL, withSessionCookie(session.ID))
		assertStatus(t, resp, http.StatusFound)
		resp.Body.Close()

		code := extractRedirectParam(t, resp, "code")
		if code == "" {
			t.Fatal("code parameter not found in redirect")
		}

		state := extractRedirectParam(t, resp, "state")
		if state != "test-state" {
			t.Errorf("state = %q, want %q", state, "test-state")
		}

		// redirect_uri が正しいホストに向いている
		loc := resp.Header.Get("Location")
		if !strings.HasPrefix(loc, redirectURI) {
			t.Errorf("redirect location %q does not start with %q", loc, redirectURI)
		}
	})

	t.Run("セッションなし -> login ページへリダイレクト", func(t *testing.T) {
		resp := doGET(t, buildAuthURL(nil))
		assertStatus(t, resp, http.StatusFound)
		resp.Body.Close()

		loc := resp.Header.Get("Location")
		if !strings.Contains(loc, "/login") {
			t.Errorf("expected redirect to login page, got: %s", loc)
		}
	})

	t.Run("無効な client_id -> 400 JSON error", func(t *testing.T) {
		resp := doGET(t, buildAuthURL(map[string]string{
			"client_id": "nonexistent-client",
		}))
		assertStatus(t, resp, http.StatusBadRequest)

		body := parseJSON(t, resp)
		if errCode, _ := body["error"].(string); errCode != "invalid_request" {
			t.Errorf("error = %q, want %q", errCode, "invalid_request")
		}
	})

	t.Run("未登録 redirect_uri -> 400 JSON error（リダイレクトしない）", func(t *testing.T) {
		resp := doGET(t, buildAuthURL(map[string]string{
			"redirect_uri": "http://evil.example.com/callback",
		}))
		assertStatus(t, resp, http.StatusBadRequest)
		body := parseJSON(t, resp)
		if errCode, _ := body["error"].(string); errCode != "invalid_request" {
			t.Errorf("error = %q, want %q", errCode, "invalid_request")
		}
	})

	t.Run("redirect_uri なし -> 400 JSON error", func(t *testing.T) {
		resp := doGET(t, buildAuthURL(map[string]string{
			"redirect_uri": "",
		}))
		assertStatus(t, resp, http.StatusBadRequest)
		resp.Body.Close()
	})

	t.Run("openid scope なし -> redirect with error=invalid_scope", func(t *testing.T) {
		resp := doGET(t, buildAuthURL(map[string]string{
			"scope": "profile",
		}), withSessionCookie(session.ID))
		assertStatus(t, resp, http.StatusFound)
		resp.Body.Close()

		errCode := extractRedirectParam(t, resp, "error")
		if errCode != "invalid_scope" {
			t.Errorf("error = %q, want %q", errCode, "invalid_scope")
		}
	})

	t.Run("PKCE 必須だが code_challenge なし -> redirect with error", func(t *testing.T) {
		resp := doGET(t, buildAuthURL(map[string]string{
			"code_challenge":        "",
			"code_challenge_method": "",
		}), withSessionCookie(session.ID))
		assertStatus(t, resp, http.StatusFound)
		resp.Body.Close()

		errCode := extractRedirectParam(t, resp, "error")
		if errCode != "invalid_request" {
			t.Errorf("error = %q, want %q", errCode, "invalid_request")
		}
	})

	t.Run("prompt=none セッションなし -> login_required", func(t *testing.T) {
		resp := doGET(t, buildAuthURL(map[string]string{
			"prompt": "none",
		}))
		assertStatus(t, resp, http.StatusFound)
		resp.Body.Close()

		errCode := extractRedirectParam(t, resp, "error")
		if errCode != "login_required" {
			t.Errorf("error = %q, want %q", errCode, "login_required")
		}
	})

	t.Run("prompt=none 同意なし -> consent_required", func(t *testing.T) {
		// 同意レコードなしの状態でテスト
		resp := doGET(t, buildAuthURL(map[string]string{
			"prompt": "none",
		}), withSessionCookie(session.ID))
		assertStatus(t, resp, http.StatusFound)
		resp.Body.Close()

		errCode := extractRedirectParam(t, resp, "error")
		if errCode != "consent_required" {
			t.Errorf("error = %q, want %q", errCode, "consent_required")
		}
	})

	t.Run("同意なし -> consent ページへリダイレクト", func(t *testing.T) {
		resp := doGET(t, buildAuthURL(nil), withSessionCookie(session.ID))
		assertStatus(t, resp, http.StatusFound)
		resp.Body.Close()

		loc := resp.Header.Get("Location")
		if !strings.Contains(loc, "/consent") {
			t.Errorf("expected redirect to consent page, got: %s", loc)
		}
	})

	t.Run("無効な response_type -> 400 error", func(t *testing.T) {
		createConsent(t, user.ID, client.ID, "openid")
		defer testDB.Exec("DELETE FROM user_consents WHERE user_id = ?", user.ID)

		resp := doGET(t, buildAuthURL(map[string]string{
			"response_type": "token",
		}), withSessionCookie(session.ID))
		// 実装では response_type チェックが redirect_uri 検証前に行われるため、
		// 直接 400 JSON エラーを返す
		assertStatus(t, resp, http.StatusBadRequest)

		body := parseJSON(t, resp)
		if errCode, _ := body["error"].(string); errCode != "unsupported_response_type" {
			t.Errorf("error = %q, want %q", errCode, "unsupported_response_type")
		}
	})

	t.Run("state が透過される", func(t *testing.T) {
		createConsent(t, user.ID, client.ID, "openid")
		defer testDB.Exec("DELETE FROM user_consents WHERE user_id = ?", user.ID)

		customState := "custom-state-" + uuid.New().String()
		resp := doGET(t, buildAuthURL(map[string]string{
			"state": customState,
		}), withSessionCookie(session.ID))
		assertStatus(t, resp, http.StatusFound)
		resp.Body.Close()

		returnedState := extractRedirectParam(t, resp, "state")
		if returnedState != customState {
			t.Errorf("state = %q, want %q", returnedState, customState)
		}
	})
}
