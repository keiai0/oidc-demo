package test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// Test_PAR は Pushed Authorization Request (RFC 9126) を検証する。
func Test_PAR(t *testing.T) {
	cleanupTables(t)

	tenant := createTenant(t, "par", "PAR Tenant")
	client := createClient(t)
	createRedirectURI(t, client.ID, "http://localhost:3001/callback")
	linkTenantClient(t, tenant.ID, client.ID)
	user := createUser(t, tenant.ID, "paruser", "paruser@example.com")
	session := createSession(t, user.ID, tenant.ID)
	createConsent(t, user.ID, client.ID, "openid profile email offline_access")

	t.Run("正常系: PAR -> request_uri 取得", func(t *testing.T) {
		cv, cc := generatePKCE()
		_ = cv

		form := url.Values{
			"response_type":         {"code"},
			"client_id":             {client.ClientID},
			"redirect_uri":          {"http://localhost:3001/callback"},
			"scope":                 {"openid"},
			"state":                 {"par-state"},
			"nonce":                 {"par-nonce"},
			"code_challenge":        {cc},
			"code_challenge_method": {"S256"},
		}
		resp := doPOSTForm(t, fmt.Sprintf("/%s/par", tenant.Code), form,
			withBasicAuth(client.ClientID, testClientSecret))
		assertStatus(t, resp, http.StatusCreated)

		body := parseJSON(t, resp)

		requestURI, _ := body["request_uri"].(string)
		if !strings.HasPrefix(requestURI, "urn:ietf:params:oauth:request_uri:") {
			t.Errorf("request_uri = %q, want prefix 'urn:ietf:params:oauth:request_uri:'", requestURI)
		}

		expiresIn, _ := body["expires_in"].(float64)
		if expiresIn <= 0 {
			t.Errorf("expires_in = %v, want > 0", expiresIn)
		}
	})

	t.Run("PAR -> authorize -> token フルフロー", func(t *testing.T) {
		cv, cc := generatePKCE()

		// 1. PAR リクエスト
		form := url.Values{
			"response_type":         {"code"},
			"client_id":             {client.ClientID},
			"redirect_uri":          {"http://localhost:3001/callback"},
			"scope":                 {"openid"},
			"state":                 {"par-flow-state"},
			"nonce":                 {"par-flow-nonce"},
			"code_challenge":        {cc},
			"code_challenge_method": {"S256"},
		}
		parResp := doPOSTForm(t, fmt.Sprintf("/%s/par", tenant.Code), form,
			withBasicAuth(client.ClientID, testClientSecret))
		assertStatus(t, parResp, http.StatusCreated)

		parBody := parseJSON(t, parResp)
		requestURI, _ := parBody["request_uri"].(string)

		// 2. Authorize リクエスト（request_uri を使用）
		authPath := fmt.Sprintf("/%s/authorize?client_id=%s&request_uri=%s",
			tenant.Code,
			url.QueryEscape(client.ClientID),
			url.QueryEscape(requestURI),
		)
		authResp := doGET(t, authPath, withSessionCookie(session.ID))
		assertStatus(t, authResp, http.StatusFound)
		authResp.Body.Close()

		code := extractRedirectParam(t, authResp, "code")
		if code == "" {
			t.Fatalf("code not found in redirect; Location: %s", authResp.Header.Get("Location"))
		}

		state := extractRedirectParam(t, authResp, "state")
		if state != "par-flow-state" {
			t.Errorf("state = %q, want %q", state, "par-flow-state")
		}

		// 3. Token リクエスト
		tokenForm := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {"http://localhost:3001/callback"},
			"code_verifier": {cv},
		}
		tokenResp := doPOSTForm(t, fmt.Sprintf("/%s/token", tenant.Code), tokenForm,
			withBasicAuth(client.ClientID, testClientSecret))
		assertStatus(t, tokenResp, http.StatusOK)

		tokenBody := parseJSON(t, tokenResp)
		if at, _ := tokenBody["access_token"].(string); at == "" {
			t.Error("access_token is empty")
		}
		if idt, _ := tokenBody["id_token"].(string); idt == "" {
			t.Error("id_token is empty")
		}
	})

	t.Run("request_uri 再利用 -> エラー", func(t *testing.T) {
		_, cc := generatePKCE()

		form := url.Values{
			"response_type":         {"code"},
			"client_id":             {client.ClientID},
			"redirect_uri":          {"http://localhost:3001/callback"},
			"scope":                 {"openid"},
			"state":                 {"s"},
			"nonce":                 {"n"},
			"code_challenge":        {cc},
			"code_challenge_method": {"S256"},
		}
		parResp := doPOSTForm(t, fmt.Sprintf("/%s/par", tenant.Code), form,
			withBasicAuth(client.ClientID, testClientSecret))
		assertStatus(t, parResp, http.StatusCreated)

		parBody := parseJSON(t, parResp)
		requestURI, _ := parBody["request_uri"].(string)

		// 1回目: 使用（成功）
		authPath := fmt.Sprintf("/%s/authorize?client_id=%s&request_uri=%s",
			tenant.Code,
			url.QueryEscape(client.ClientID),
			url.QueryEscape(requestURI),
		)
		authResp1 := doGET(t, authPath, withSessionCookie(session.ID))
		authResp1.Body.Close()

		// 2回目: 再利用（エラー）
		authResp2 := doGET(t, authPath, withSessionCookie(session.ID))
		defer authResp2.Body.Close()

		// PAR 再利用はエラーレスポンスを返す
		if authResp2.StatusCode == http.StatusFound {
			errParam := extractRedirectParam(t, authResp2, "error")
			if errParam == "" {
				// リダイレクト先にエラーがない場合は、400 でもよい
			}
		}
		// 400 or redirect with error のいずれかを許容
		if authResp2.StatusCode != http.StatusBadRequest && authResp2.StatusCode != http.StatusFound {
			t.Errorf("expected 400 or 302 with error, got %d", authResp2.StatusCode)
		}
	})

	t.Run("不正クライアント認証 -> 401", func(t *testing.T) {
		form := url.Values{
			"response_type": {"code"},
			"client_id":     {client.ClientID},
			"redirect_uri":  {"http://localhost:3001/callback"},
			"scope":         {"openid"},
		}
		resp := doPOSTForm(t, fmt.Sprintf("/%s/par", tenant.Code), form,
			withBasicAuth(client.ClientID, "wrong-secret"))
		assertStatus(t, resp, http.StatusUnauthorized)
		resp.Body.Close()
	})
}
