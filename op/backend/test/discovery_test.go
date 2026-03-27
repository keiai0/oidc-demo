package test

import (
	"net/http"
	"testing"
)

// Test_Discovery は OIDC Discovery エンドポイント（/.well-known/openid-configuration）を検証する。
// OpenID Foundation 認定の「Config OP」プロファイルに相当。
func Test_Discovery(t *testing.T) {
	cleanupTables(t)
	tenant := createTenant(t, "disc", "Discovery Tenant")

	t.Run("必須フィールドが存在する", func(t *testing.T) {
		resp := doGET(t, "/disc/.well-known/openid-configuration")
		assertStatus(t, resp, http.StatusOK)

		body := parseJSON(t, resp)

		requiredFields := []string{
			"issuer",
			"authorization_endpoint",
			"token_endpoint",
			"userinfo_endpoint",
			"jwks_uri",
			"response_types_supported",
			"subject_types_supported",
			"id_token_signing_alg_values_supported",
		}
		for _, field := range requiredFields {
			if _, ok := body[field]; !ok {
				t.Errorf("missing required field: %s", field)
			}
		}
	})

	t.Run("issuer が正しい形式", func(t *testing.T) {
		resp := doGET(t, "/disc/.well-known/openid-configuration")
		body := parseJSON(t, resp)

		issuer, _ := body["issuer"].(string)
		expected := serverURL + "/" + tenant.Code
		if issuer != expected {
			t.Errorf("issuer = %q, want %q", issuer, expected)
		}

		// 末尾スラッシュがないこと（OIDC Discovery Section 4.3）
		if issuer != "" && issuer[len(issuer)-1] == '/' {
			t.Error("issuer must not end with a trailing slash")
		}
	})

	t.Run("エンドポイント URL が serverURL で始まる", func(t *testing.T) {
		resp := doGET(t, "/disc/.well-known/openid-configuration")
		body := parseJSON(t, resp)

		endpointFields := []string{
			"authorization_endpoint",
			"token_endpoint",
			"userinfo_endpoint",
			"jwks_uri",
			"revocation_endpoint",
			"introspection_endpoint",
		}
		for _, field := range endpointFields {
			val, _ := body[field].(string)
			if val == "" {
				t.Errorf("%s is empty", field)
				continue
			}
			if len(val) < len(serverURL) || val[:len(serverURL)] != serverURL {
				t.Errorf("%s = %q, should start with %q", field, val, serverURL)
			}
		}
	})

	t.Run("Cache-Control ヘッダ", func(t *testing.T) {
		resp := doGET(t, "/disc/.well-known/openid-configuration")
		defer resp.Body.Close()

		cc := resp.Header.Get("Cache-Control")
		if cc != "public, max-age=86400" {
			t.Errorf("Cache-Control = %q, want %q", cc, "public, max-age=86400")
		}
	})

	t.Run("サポートされるメタデータ値", func(t *testing.T) {
		resp := doGET(t, "/disc/.well-known/openid-configuration")
		body := parseJSON(t, resp)

		// response_types_supported に "code" が含まれる
		rtSupported, _ := body["response_types_supported"].([]any)
		if !containsValue(rtSupported, "code") {
			t.Error("response_types_supported should contain 'code'")
		}

		// subject_types_supported
		stSupported, _ := body["subject_types_supported"].([]any)
		if !containsValue(stSupported, "public") {
			t.Error("subject_types_supported should contain 'public'")
		}

		// id_token_signing_alg_values_supported に "RS256"
		algSupported, _ := body["id_token_signing_alg_values_supported"].([]any)
		if !containsValue(algSupported, "RS256") {
			t.Error("id_token_signing_alg_values_supported should contain 'RS256'")
		}

		// code_challenge_methods_supported に "S256"
		ccmSupported, _ := body["code_challenge_methods_supported"].([]any)
		if !containsValue(ccmSupported, "S256") {
			t.Error("code_challenge_methods_supported should contain 'S256'")
		}

		// token_endpoint_auth_methods_supported
		authMethods, _ := body["token_endpoint_auth_methods_supported"].([]any)
		if !containsValue(authMethods, "client_secret_basic") {
			t.Error("token_endpoint_auth_methods_supported should contain 'client_secret_basic'")
		}
		if !containsValue(authMethods, "client_secret_post") {
			t.Error("token_endpoint_auth_methods_supported should contain 'client_secret_post'")
		}

		// scopes_supported に "openid"
		scopesSupported, _ := body["scopes_supported"].([]any)
		if !containsValue(scopesSupported, "openid") {
			t.Error("scopes_supported should contain 'openid'")
		}

		// grant_types_supported
		gtSupported, _ := body["grant_types_supported"].([]any)
		for _, gt := range []string{"authorization_code", "refresh_token", "client_credentials"} {
			if !containsValue(gtSupported, gt) {
				t.Errorf("grant_types_supported should contain %q", gt)
			}
		}
	})

	t.Run("存在しないテナント -> 404", func(t *testing.T) {
		resp := doGET(t, "/nonexistent/.well-known/openid-configuration")
		assertStatus(t, resp, http.StatusNotFound)
		resp.Body.Close()
	})
}

// Test_JWKS は JWKS エンドポイント（/jwks）を検証する。
func Test_JWKS(t *testing.T) {
	t.Run("JWK Set のフォーマット検証", func(t *testing.T) {
		resp := doGET(t, "/jwks")
		assertStatus(t, resp, http.StatusOK)

		body := parseJSON(t, resp)

		keys, ok := body["keys"].([]any)
		if !ok || len(keys) == 0 {
			t.Fatal("JWKS should have non-empty 'keys' array")
		}

		for i, k := range keys {
			key, ok := k.(map[string]any)
			if !ok {
				t.Fatalf("keys[%d] is not an object", i)
			}

			requiredFields := []string{"kid", "kty", "alg", "use"}
			for _, field := range requiredFields {
				if _, exists := key[field]; !exists {
					t.Errorf("keys[%d] missing field: %s", i, field)
				}
			}

			// kty は "RSA" であること
			if kty, _ := key["kty"].(string); kty != "RSA" {
				t.Errorf("keys[%d] kty = %q, want %q", i, kty, "RSA")
			}

			// alg は "RS256" であること
			if alg, _ := key["alg"].(string); alg != "RS256" {
				t.Errorf("keys[%d] alg = %q, want %q", i, alg, "RS256")
			}

			// use は "sig" であること
			if use, _ := key["use"].(string); use != "sig" {
				t.Errorf("keys[%d] use = %q, want %q", i, use, "sig")
			}
		}
	})
}

// containsValue は any スライスに指定した文字列が含まれるか確認する。
func containsValue(slice []any, target string) bool {
	for _, v := range slice {
		if s, ok := v.(string); ok && s == target {
			return true
		}
	}
	return false
}
