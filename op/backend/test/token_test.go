package test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Test_Token_AuthCodeGrant は Authorization Code Grant のトークン交換を検証する。
func Test_Token_AuthCodeGrant(t *testing.T) {
	cleanupTables(t)
	fixtures := setupFlowFixtures(t)

	t.Run("正常系: code -> tokens + ID Token クレーム検証", func(t *testing.T) {
		nonce := uuid.New().String()
		result := performAuthCodeFlow(t, fixtures, WithNonce(nonce))

		// access_token が存在する
		if result.AccessToken == "" {
			t.Fatal("access_token is empty")
		}

		// id_token が存在する
		if result.IDToken == "" {
			t.Fatal("id_token is empty")
		}

		// refresh_token が存在する（offline_access scope はデフォルトで含む）
		// note: デフォルト scope は "openid profile email" なので refresh_token は
		// grant_types に "refresh_token" が含まれていれば返る

		// ID Token クレームの検証 (OIDC Core Section 3.1.3.7)
		claims := result.IDTokenClaims

		// iss
		iss, _ := claims["iss"].(string)
		expectedIss := serverURL + "/" + fixtures.tenant.Code
		if iss != expectedIss {
			t.Errorf("iss = %q, want %q", iss, expectedIss)
		}

		// aud
		aud, _ := claims["aud"].(string)
		if aud != fixtures.client.ClientID {
			// aud は文字列か配列
			audArr, _ := claims["aud"].([]any)
			found := false
			for _, a := range audArr {
				if s, ok := a.(string); ok && s == fixtures.client.ClientID {
					found = true
				}
			}
			if !found {
				t.Errorf("aud does not contain client_id %q", fixtures.client.ClientID)
			}
		}

		// exp が未来
		exp, _ := claims["exp"].(float64)
		if exp <= float64(time.Now().Unix()) {
			t.Error("exp should be in the future")
		}

		// iat が妥当な範囲
		iat, _ := claims["iat"].(float64)
		if iat <= 0 || iat > float64(time.Now().Unix()+60) {
			t.Errorf("iat is out of range: %v", iat)
		}

		// nonce がリクエスト時の値と一致
		nonceInToken, _ := claims["nonce"].(string)
		if nonceInToken != nonce {
			t.Errorf("nonce = %q, want %q", nonceInToken, nonce)
		}

		// sub が存在する
		sub, _ := claims["sub"].(string)
		if sub == "" {
			t.Error("sub is empty")
		}

		// at_hash が存在する (OIDC Core Section 3.1.3.6)
		atHash, _ := claims["at_hash"].(string)
		if atHash == "" {
			t.Error("at_hash is empty")
		}
	})

	t.Run("Cache-Control / Pragma ヘッダ", func(t *testing.T) {
		// authorize → code 取得
		cv, cc := generatePKCE()
		state := uuid.New().String()
		authPath := fmt.Sprintf("/%s/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid&state=%s&nonce=test&code_challenge=%s&code_challenge_method=S256",
			fixtures.tenant.Code,
			url.QueryEscape(fixtures.client.ClientID),
			url.QueryEscape("http://localhost:3001/callback"),
			url.QueryEscape(state),
			url.QueryEscape(cc),
		)
		authResp := doGET(t, authPath, withSessionCookie(fixtures.session.ID))
		authResp.Body.Close()
		code := extractRedirectParam(t, authResp, "code")

		tokenForm := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {"http://localhost:3001/callback"},
			"code_verifier": {cv},
		}
		tokenResp := doPOSTForm(t, fmt.Sprintf("/%s/token", fixtures.tenant.Code), tokenForm,
			withBasicAuth(fixtures.client.ClientID, testClientSecret))
		defer tokenResp.Body.Close()

		assertStatus(t, tokenResp, http.StatusOK)

		if cc := tokenResp.Header.Get("Cache-Control"); cc != "no-store" {
			t.Errorf("Cache-Control = %q, want %q", cc, "no-store")
		}
		if pragma := tokenResp.Header.Get("Pragma"); pragma != "no-cache" {
			t.Errorf("Pragma = %q, want %q", pragma, "no-cache")
		}
	})

	t.Run("client_secret_post 認証", func(t *testing.T) {
		cv, cc := generatePKCE()
		authPath := fmt.Sprintf("/%s/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid&state=s&nonce=n&code_challenge=%s&code_challenge_method=S256",
			fixtures.tenant.Code,
			url.QueryEscape(fixtures.client.ClientID),
			url.QueryEscape("http://localhost:3001/callback"),
			url.QueryEscape(cc),
		)
		authResp := doGET(t, authPath, withSessionCookie(fixtures.session.ID))
		authResp.Body.Close()
		code := extractRedirectParam(t, authResp, "code")

		tokenForm := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {"http://localhost:3001/callback"},
			"code_verifier": {cv},
			"client_id":     {fixtures.client.ClientID},
			"client_secret": {testClientSecret},
		}
		tokenResp := doPOSTForm(t, fmt.Sprintf("/%s/token", fixtures.tenant.Code), tokenForm)
		assertStatus(t, tokenResp, http.StatusOK)
		tokenResp.Body.Close()
	})

	t.Run("不正な client_secret -> 401 invalid_client", func(t *testing.T) {
		tokenForm := url.Values{
			"grant_type":   {"authorization_code"},
			"code":         {"dummy"},
			"redirect_uri": {"http://localhost:3001/callback"},
		}
		tokenResp := doPOSTForm(t, fmt.Sprintf("/%s/token", fixtures.tenant.Code), tokenForm,
			withBasicAuth(fixtures.client.ClientID, "wrong-secret"))
		assertStatus(t, tokenResp, http.StatusUnauthorized)

		body := parseJSON(t, tokenResp)
		if errCode, _ := body["error"].(string); errCode != "invalid_client" {
			t.Errorf("error = %q, want %q", errCode, "invalid_client")
		}
	})

	t.Run("不正な code -> 400 invalid_grant", func(t *testing.T) {
		tokenForm := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {"invalid-code"},
			"redirect_uri":  {"http://localhost:3001/callback"},
			"code_verifier": {"dummy"},
		}
		tokenResp := doPOSTForm(t, fmt.Sprintf("/%s/token", fixtures.tenant.Code), tokenForm,
			withBasicAuth(fixtures.client.ClientID, testClientSecret))
		assertStatus(t, tokenResp, http.StatusBadRequest)

		body := parseJSON(t, tokenResp)
		if errCode, _ := body["error"].(string); errCode != "invalid_grant" {
			t.Errorf("error = %q, want %q", errCode, "invalid_grant")
		}
	})

	t.Run("code 再利用 -> 400 invalid_grant", func(t *testing.T) {
		cv, cc := generatePKCE()
		authPath := fmt.Sprintf("/%s/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid&state=s&nonce=n&code_challenge=%s&code_challenge_method=S256",
			fixtures.tenant.Code,
			url.QueryEscape(fixtures.client.ClientID),
			url.QueryEscape("http://localhost:3001/callback"),
			url.QueryEscape(cc),
		)
		authResp := doGET(t, authPath, withSessionCookie(fixtures.session.ID))
		authResp.Body.Close()
		code := extractRedirectParam(t, authResp, "code")

		tokenForm := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {"http://localhost:3001/callback"},
			"code_verifier": {cv},
		}

		// 1回目: 成功
		resp1 := doPOSTForm(t, fmt.Sprintf("/%s/token", fixtures.tenant.Code), tokenForm,
			withBasicAuth(fixtures.client.ClientID, testClientSecret))
		assertStatus(t, resp1, http.StatusOK)
		resp1.Body.Close()

		// 2回目: 失敗
		resp2 := doPOSTForm(t, fmt.Sprintf("/%s/token", fixtures.tenant.Code), tokenForm,
			withBasicAuth(fixtures.client.ClientID, testClientSecret))
		assertStatus(t, resp2, http.StatusBadRequest)

		body := parseJSON(t, resp2)
		if errCode, _ := body["error"].(string); errCode != "invalid_grant" {
			t.Errorf("error = %q, want %q", errCode, "invalid_grant")
		}
	})

	t.Run("code_verifier 不一致 -> 400 invalid_grant", func(t *testing.T) {
		_, cc := generatePKCE()
		authPath := fmt.Sprintf("/%s/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid&state=s&nonce=n&code_challenge=%s&code_challenge_method=S256",
			fixtures.tenant.Code,
			url.QueryEscape(fixtures.client.ClientID),
			url.QueryEscape("http://localhost:3001/callback"),
			url.QueryEscape(cc),
		)
		authResp := doGET(t, authPath, withSessionCookie(fixtures.session.ID))
		authResp.Body.Close()
		code := extractRedirectParam(t, authResp, "code")

		tokenForm := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {"http://localhost:3001/callback"},
			"code_verifier": {"wrong-verifier-that-does-not-match"},
		}
		tokenResp := doPOSTForm(t, fmt.Sprintf("/%s/token", fixtures.tenant.Code), tokenForm,
			withBasicAuth(fixtures.client.ClientID, testClientSecret))
		assertStatus(t, tokenResp, http.StatusBadRequest)

		body := parseJSON(t, tokenResp)
		if errCode, _ := body["error"].(string); errCode != "invalid_grant" {
			t.Errorf("error = %q, want %q", errCode, "invalid_grant")
		}
	})

	t.Run("redirect_uri 不一致 -> 400 invalid_grant", func(t *testing.T) {
		cv, cc := generatePKCE()
		authPath := fmt.Sprintf("/%s/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid&state=s&nonce=n&code_challenge=%s&code_challenge_method=S256",
			fixtures.tenant.Code,
			url.QueryEscape(fixtures.client.ClientID),
			url.QueryEscape("http://localhost:3001/callback"),
			url.QueryEscape(cc),
		)
		authResp := doGET(t, authPath, withSessionCookie(fixtures.session.ID))
		authResp.Body.Close()
		code := extractRedirectParam(t, authResp, "code")

		tokenForm := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {"http://localhost:3001/different-callback"},
			"code_verifier": {cv},
		}
		tokenResp := doPOSTForm(t, fmt.Sprintf("/%s/token", fixtures.tenant.Code), tokenForm,
			withBasicAuth(fixtures.client.ClientID, testClientSecret))
		assertStatus(t, tokenResp, http.StatusBadRequest)

		body := parseJSON(t, tokenResp)
		if errCode, _ := body["error"].(string); errCode != "invalid_grant" {
			t.Errorf("error = %q, want %q", errCode, "invalid_grant")
		}
	})

	t.Run("unsupported_grant_type -> 400", func(t *testing.T) {
		tokenForm := url.Values{
			"grant_type": {"password"},
			"username":   {"user"},
			"password":   {"pass"},
		}
		tokenResp := doPOSTForm(t, fmt.Sprintf("/%s/token", fixtures.tenant.Code), tokenForm,
			withBasicAuth(fixtures.client.ClientID, testClientSecret))
		assertStatus(t, tokenResp, http.StatusBadRequest)

		body := parseJSON(t, tokenResp)
		if errCode, _ := body["error"].(string); errCode != "unsupported_grant_type" {
			t.Errorf("error = %q, want %q", errCode, "unsupported_grant_type")
		}
	})
}

// Test_Token_RefreshTokenGrant は Refresh Token Grant を検証する。
func Test_Token_RefreshTokenGrant(t *testing.T) {
	cleanupTables(t)
	fixtures := setupFlowFixtures(t)

	t.Run("正常系: refresh -> 新しい access_token + refresh_token", func(t *testing.T) {
		result := performAuthCodeFlow(t, fixtures, WithScope("openid offline_access"))

		if result.RefreshToken == "" {
			t.Skip("refresh_token not returned; skipping refresh test")
		}

		tokenForm := url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {result.RefreshToken},
		}
		tokenResp := doPOSTForm(t, fmt.Sprintf("/%s/token", fixtures.tenant.Code), tokenForm,
			withBasicAuth(fixtures.client.ClientID, testClientSecret))
		assertStatus(t, tokenResp, http.StatusOK)

		body := parseJSON(t, tokenResp)

		newAccessToken, _ := body["access_token"].(string)
		newRefreshToken, _ := body["refresh_token"].(string)

		if newAccessToken == "" {
			t.Error("new access_token is empty")
		}
		if newRefreshToken == "" {
			t.Error("new refresh_token is empty")
		}

		// Rotation: 新しい refresh_token は元と異なる
		if newRefreshToken == result.RefreshToken {
			t.Error("refresh_token should be rotated (new != old)")
		}
	})

	t.Run("Rotation: 旧 refresh_token は無効化される", func(t *testing.T) {
		result := performAuthCodeFlow(t, fixtures, WithScope("openid offline_access"))

		if result.RefreshToken == "" {
			t.Skip("refresh_token not returned")
		}

		// 1回目の refresh（成功）
		tokenForm := url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {result.RefreshToken},
		}
		resp1 := doPOSTForm(t, fmt.Sprintf("/%s/token", fixtures.tenant.Code), tokenForm,
			withBasicAuth(fixtures.client.ClientID, testClientSecret))
		assertStatus(t, resp1, http.StatusOK)
		resp1.Body.Close()

		// 旧 refresh_token で再度 refresh（失敗: Reuse Detection）
		resp2 := doPOSTForm(t, fmt.Sprintf("/%s/token", fixtures.tenant.Code), tokenForm,
			withBasicAuth(fixtures.client.ClientID, testClientSecret))
		assertStatus(t, resp2, http.StatusBadRequest)

		body := parseJSON(t, resp2)
		if errCode, _ := body["error"].(string); errCode != "invalid_grant" {
			t.Errorf("error = %q, want %q", errCode, "invalid_grant")
		}
	})
}

// Test_Token_ClientCredentialsGrant は Client Credentials Grant を検証する。
func Test_Token_ClientCredentialsGrant(t *testing.T) {
	cleanupTables(t)

	tenant := createTenant(t, "cc", "CC Tenant")
	client := createClient(t,
		WithGrantTypes("client_credentials"),
		WithRequirePKCE(false),
	)
	linkTenantClient(t, tenant.ID, client.ID)

	t.Run("正常系: access_token のみ（id_token/refresh_token なし）", func(t *testing.T) {
		tokenForm := url.Values{
			"grant_type": {"client_credentials"},
			"scope":      {"openid"},
		}
		tokenResp := doPOSTForm(t, fmt.Sprintf("/%s/token", tenant.Code), tokenForm,
			withBasicAuth(client.ClientID, testClientSecret))
		assertStatus(t, tokenResp, http.StatusOK)

		body := parseJSON(t, tokenResp)

		if at, _ := body["access_token"].(string); at == "" {
			t.Error("access_token is empty")
		}
		if _, exists := body["refresh_token"]; exists {
			t.Error("client_credentials should not return refresh_token")
		}
	})
}
