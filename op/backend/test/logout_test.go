package test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// Test_Logout は RP-Initiated Logout を検証する。
func Test_Logout(t *testing.T) {
	cleanupTables(t)

	tenant := createTenant(t, "logout", "Logout Tenant")
	client := createClient(t)
	createRedirectURI(t, client.ID, "http://localhost:3001/callback")
	createPostLogoutRedirectURI(t, client.ID, "http://localhost:3001/logged-out")
	linkTenantClient(t, tenant.ID, client.ID)
	user := createUser(t, tenant.ID, "logoutuser", "logoutuser@example.com")

	t.Run("id_token_hint 付きログアウト -> post_logout_redirect_uri へリダイレクト", func(t *testing.T) {
		testDB.Exec("DELETE FROM user_consents WHERE user_id = ?", user.ID)
		session := createSession(t, user.ID, tenant.ID)
		createConsent(t, user.ID, client.ID, "openid profile email offline_access")

		fixtures := &flowFixtures{tenant: tenant, client: client, user: user, session: session}
		result := performAuthCodeFlow(t, fixtures)

		// ログアウトリクエスト
		logoutPath := fmt.Sprintf("/%s/logout?id_token_hint=%s&post_logout_redirect_uri=%s&state=logout-state",
			tenant.Code,
			url.QueryEscape(result.IDToken),
			url.QueryEscape("http://localhost:3001/logged-out"),
		)
		resp := doGET(t, logoutPath, withSessionCookie(session.ID))
		defer resp.Body.Close()

		// 302 リダイレクト
		assertStatus(t, resp, http.StatusFound)

		loc := resp.Header.Get("Location")
		if !strings.HasPrefix(loc, "http://localhost:3001/logged-out") {
			t.Errorf("expected redirect to post_logout_redirect_uri, got: %s", loc)
		}

		// state が透過される
		u, _ := url.Parse(loc)
		if s := u.Query().Get("state"); s != "logout-state" {
			t.Errorf("state = %q, want %q", s, "logout-state")
		}
	})

	t.Run("id_token_hint なし（Cookie ベース）-> ログアウト成功", func(t *testing.T) {
		session := createSession(t, user.ID, tenant.ID)

		logoutPath := fmt.Sprintf("/%s/logout", tenant.Code)
		resp := doGET(t, logoutPath, withSessionCookie(session.ID))
		defer resp.Body.Close()

		// セッションが失効されて何らかのリダイレクトが返る
		assertStatus(t, resp, http.StatusFound)
	})

	t.Run("未登録 post_logout_redirect_uri -> デフォルトリダイレクト", func(t *testing.T) {
		testDB.Exec("DELETE FROM user_consents WHERE user_id = ?", user.ID)
		session := createSession(t, user.ID, tenant.ID)
		createConsent(t, user.ID, client.ID, "openid profile email offline_access")

		fixtures := &flowFixtures{tenant: tenant, client: client, user: user, session: session}
		result := performAuthCodeFlow(t, fixtures)

		logoutPath := fmt.Sprintf("/%s/logout?id_token_hint=%s&post_logout_redirect_uri=%s",
			tenant.Code,
			url.QueryEscape(result.IDToken),
			url.QueryEscape("http://evil.example.com/logged-out"),
		)
		resp := doGET(t, logoutPath, withSessionCookie(session.ID))
		defer resp.Body.Close()

		assertStatus(t, resp, http.StatusFound)

		loc := resp.Header.Get("Location")
		// evil.example.com にリダイレクトされないこと
		if strings.Contains(loc, "evil.example.com") {
			t.Errorf("should not redirect to unregistered URI, got: %s", loc)
		}
	})
}
