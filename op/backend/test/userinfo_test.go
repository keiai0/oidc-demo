package test

import (
	"net/http"
	"testing"
)

// Test_UserInfo は UserInfo エンドポイントを検証する。
func Test_UserInfo(t *testing.T) {
	cleanupTables(t)
	fixtures := setupFlowFixtures(t)

	t.Run("正常系: 有効な access_token で sub が返る", func(t *testing.T) {
		result := performAuthCodeFlow(t, fixtures)

		resp := doGET(t, "/"+fixtures.tenant.Code+"/userinfo",
			withBearerToken(result.AccessToken))
		assertStatus(t, resp, http.StatusOK)

		body := parseJSON(t, resp)

		sub, _ := body["sub"].(string)
		if sub == "" {
			t.Error("sub is empty")
		}

		// sub が ID Token の sub と一致する
		idTokenSub, _ := result.IDTokenClaims["sub"].(string)
		if sub != idTokenSub {
			t.Errorf("userinfo sub = %q, id_token sub = %q", sub, idTokenSub)
		}
	})

	t.Run("scope=profile -> name が含まれる", func(t *testing.T) {
		result := performAuthCodeFlow(t, fixtures, WithScope("openid profile"))

		resp := doGET(t, "/"+fixtures.tenant.Code+"/userinfo",
			withBearerToken(result.AccessToken))
		assertStatus(t, resp, http.StatusOK)

		body := parseJSON(t, resp)

		if _, ok := body["name"]; !ok {
			t.Error("name should be present with profile scope")
		}
	})

	t.Run("scope=email -> email, email_verified が含まれる", func(t *testing.T) {
		result := performAuthCodeFlow(t, fixtures, WithScope("openid email"))

		resp := doGET(t, "/"+fixtures.tenant.Code+"/userinfo",
			withBearerToken(result.AccessToken))
		assertStatus(t, resp, http.StatusOK)

		body := parseJSON(t, resp)

		if _, ok := body["email"]; !ok {
			t.Error("email should be present with email scope")
		}
		if _, ok := body["email_verified"]; !ok {
			t.Error("email_verified should be present with email scope")
		}
	})

	t.Run("無効なトークン -> 401", func(t *testing.T) {
		resp := doGET(t, "/"+fixtures.tenant.Code+"/userinfo",
			withBearerToken("invalid-token"))
		assertStatus(t, resp, http.StatusUnauthorized)
		resp.Body.Close()
	})

	t.Run("Authorization ヘッダなし -> 401", func(t *testing.T) {
		resp := doGET(t, "/"+fixtures.tenant.Code+"/userinfo")
		assertStatus(t, resp, http.StatusUnauthorized)
		resp.Body.Close()
	})
}
