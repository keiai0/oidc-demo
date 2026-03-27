package test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

// Test_Revoke はトークン失効（RFC 7009）を検証する。
func Test_Revoke(t *testing.T) {
	cleanupTables(t)
	fixtures := setupFlowFixtures(t)

	t.Run("access_token 失効 -> 200 OK", func(t *testing.T) {
		result := performAuthCodeFlow(t, fixtures)

		form := url.Values{
			"token":           {result.AccessToken},
			"token_type_hint": {"access_token"},
		}
		resp := doPOSTForm(t, fmt.Sprintf("/%s/revoke", fixtures.tenant.Code), form,
			withBasicAuth(fixtures.client.ClientID, testClientSecret))
		assertStatus(t, resp, http.StatusOK)
		resp.Body.Close()
	})

	t.Run("refresh_token 失効 -> 200 OK", func(t *testing.T) {
		result := performAuthCodeFlow(t, fixtures, WithScope("openid offline_access"))

		if result.RefreshToken == "" {
			t.Skip("refresh_token not returned")
		}

		form := url.Values{
			"token":           {result.RefreshToken},
			"token_type_hint": {"refresh_token"},
		}
		resp := doPOSTForm(t, fmt.Sprintf("/%s/revoke", fixtures.tenant.Code), form,
			withBasicAuth(fixtures.client.ClientID, testClientSecret))
		assertStatus(t, resp, http.StatusOK)
		resp.Body.Close()
	})

	t.Run("存在しないトークン -> 200 OK (RFC 7009 Section 2.2)", func(t *testing.T) {
		form := url.Values{
			"token": {"nonexistent-token-value"},
		}
		resp := doPOSTForm(t, fmt.Sprintf("/%s/revoke", fixtures.tenant.Code), form,
			withBasicAuth(fixtures.client.ClientID, testClientSecret))
		assertStatus(t, resp, http.StatusOK)
		resp.Body.Close()
	})

	t.Run("不正クライアント認証 -> 401", func(t *testing.T) {
		form := url.Values{
			"token": {"some-token"},
		}
		resp := doPOSTForm(t, fmt.Sprintf("/%s/revoke", fixtures.tenant.Code), form,
			withBasicAuth(fixtures.client.ClientID, "wrong-secret"))
		assertStatus(t, resp, http.StatusUnauthorized)
		resp.Body.Close()
	})

	t.Run("失効後の access_token は introspection で active:false", func(t *testing.T) {
		result := performAuthCodeFlow(t, fixtures)

		// 失効
		revokeForm := url.Values{
			"token":           {result.AccessToken},
			"token_type_hint": {"access_token"},
		}
		revokeResp := doPOSTForm(t, fmt.Sprintf("/%s/revoke", fixtures.tenant.Code), revokeForm,
			withBasicAuth(fixtures.client.ClientID, testClientSecret))
		assertStatus(t, revokeResp, http.StatusOK)
		revokeResp.Body.Close()

		// introspection で確認
		introspectForm := url.Values{
			"token": {result.AccessToken},
		}
		introspectResp := doPOSTForm(t, fmt.Sprintf("/%s/introspect", fixtures.tenant.Code), introspectForm,
			withBasicAuth(fixtures.client.ClientID, testClientSecret))
		assertStatus(t, introspectResp, http.StatusOK)

		body := parseJSON(t, introspectResp)
		active, _ := body["active"].(bool)
		if active {
			t.Error("revoked token should have active=false")
		}
	})
}

// Test_Introspect はトークンイントロスペクション（RFC 7662）を検証する。
func Test_Introspect(t *testing.T) {
	cleanupTables(t)
	fixtures := setupFlowFixtures(t)

	t.Run("有効な access_token -> active:true + claims", func(t *testing.T) {
		result := performAuthCodeFlow(t, fixtures)

		form := url.Values{
			"token": {result.AccessToken},
		}
		resp := doPOSTForm(t, fmt.Sprintf("/%s/introspect", fixtures.tenant.Code), form,
			withBasicAuth(fixtures.client.ClientID, testClientSecret))
		assertStatus(t, resp, http.StatusOK)

		body := parseJSON(t, resp)
		active, _ := body["active"].(bool)
		if !active {
			t.Error("valid token should have active=true")
		}

		// claims が含まれる
		if _, ok := body["client_id"]; !ok {
			t.Error("client_id should be present")
		}
		if _, ok := body["sub"]; !ok {
			t.Error("sub should be present")
		}
	})

	t.Run("無効なトークン -> active:false", func(t *testing.T) {
		form := url.Values{
			"token": {"invalid-token-value"},
		}
		resp := doPOSTForm(t, fmt.Sprintf("/%s/introspect", fixtures.tenant.Code), form,
			withBasicAuth(fixtures.client.ClientID, testClientSecret))
		assertStatus(t, resp, http.StatusOK)

		body := parseJSON(t, resp)
		active, _ := body["active"].(bool)
		if active {
			t.Error("invalid token should have active=false")
		}
	})

	t.Run("不正クライアント認証 -> 401", func(t *testing.T) {
		form := url.Values{
			"token": {"some-token"},
		}
		resp := doPOSTForm(t, fmt.Sprintf("/%s/introspect", fixtures.tenant.Code), form,
			withBasicAuth(fixtures.client.ClientID, "wrong-secret"))
		assertStatus(t, resp, http.StatusUnauthorized)
		resp.Body.Close()
	})
}
