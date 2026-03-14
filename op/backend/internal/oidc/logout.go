package oidc

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// LogoutHandler は GET /{tenant_code}/logout (RP-Initiated Logout) を処理する。
// 仕様参照: RP-Initiated Logout 1.0
type LogoutHandler struct {
	tenantFinder               TenantFinder
	clientFinder               ClientFinder
	tenantClientChecker        TenantClientChecker
	clientsByTenantLister      ClientsByTenantLister
	postLogoutRedirectURILister PostLogoutRedirectURILister
	idTokenHintValidator       IDTokenHintValidator
	logoutTokenSigner          LogoutTokenSigner
	sessionRevoker             SessionRevoker
	accessTokenStore           AccessTokenStore
	refreshTokenStore          RefreshTokenStore
	issuerBaseURL              string
	frontendBaseURL            string
	httpClient                 *http.Client
	isSecure                   bool
}

// NewLogoutHandler は LogoutHandler を生成する。
func NewLogoutHandler(
	tenantFinder TenantFinder,
	clientFinder ClientFinder,
	tenantClientChecker TenantClientChecker,
	clientsByTenantLister ClientsByTenantLister,
	postLogoutRedirectURILister PostLogoutRedirectURILister,
	idTokenHintValidator IDTokenHintValidator,
	logoutTokenSigner LogoutTokenSigner,
	sessionRevoker SessionRevoker,
	accessTokenStore AccessTokenStore,
	refreshTokenStore RefreshTokenStore,
	issuerBaseURL string,
	frontendBaseURL string,
	httpClient *http.Client,
	isSecure bool,
) *LogoutHandler {
	return &LogoutHandler{
		tenantFinder:               tenantFinder,
		clientFinder:               clientFinder,
		tenantClientChecker:        tenantClientChecker,
		clientsByTenantLister:      clientsByTenantLister,
		postLogoutRedirectURILister: postLogoutRedirectURILister,
		idTokenHintValidator:       idTokenHintValidator,
		logoutTokenSigner:          logoutTokenSigner,
		sessionRevoker:             sessionRevoker,
		accessTokenStore:           accessTokenStore,
		refreshTokenStore:          refreshTokenStore,
		issuerBaseURL:              issuerBaseURL,
		frontendBaseURL:            frontendBaseURL,
		httpClient:                 httpClient,
		isSecure:                   isSecure,
	}
}

// Handle は GET /{tenant_code}/logout を処理する。
// 仕様参照: RP-Initiated Logout 1.0 Section 2
func (h *LogoutHandler) Handle(c echo.Context) error {
	ctx := c.Request().Context()
	tenantCode := c.Param("tenant_code")

	// テナント検証
	tenant, err := h.tenantFinder.FindByCode(ctx, tenantCode)
	if err != nil {
		c.Logger().Errorf("logout: failed to find tenant: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	if tenant == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not_found"})
	}

	issuer := h.issuerBaseURL + "/" + tenantCode

	// パラメータ取得 (GET: クエリパラメータ, POST: フォームボディ)
	// Echo の FormValue は GET/POST 両方に対応する
	idTokenHint := c.FormValue("id_token_hint")
	postLogoutRedirectURI := c.FormValue("post_logout_redirect_uri")
	state := c.FormValue("state")

	var hintResult *model.IDTokenHintResult
	var client *model.Client
	var sessionID uuid.UUID
	var userSub string

	// id_token_hint 検証
	if idTokenHint != "" {
		hintResult, err = h.idTokenHintValidator.ValidateIDTokenHint(ctx, idTokenHint)
		if err != nil {
			c.Logger().Warnf("logout: invalid id_token_hint: %v", err)
			// 仕様: id_token_hint が無効でもエラーにはしない（冪等性）
			// ただし post_logout_redirect_uri のリダイレクトは行わない
			hintResult = nil
		}

		if hintResult != nil {
			// iss 検証
			if hintResult.Issuer != issuer {
				c.Logger().Warnf("logout: id_token_hint iss mismatch: got %s, expected %s", hintResult.Issuer, issuer)
				hintResult = nil
			}
		}

		if hintResult != nil {
			// aud から client 取得
			client, err = h.clientFinder.FindByClientID(ctx, hintResult.Audience)
			if err != nil {
				c.Logger().Errorf("logout: failed to find client: %v", err)
			}
			if client == nil || client.Status != "active" {
				c.Logger().Warnf("logout: client not found or inactive for aud=%s", hintResult.Audience)
				hintResult = nil
				client = nil
			}
		}

		if hintResult != nil && client != nil {
			// テナント-クライアント紐付き検証
			exists, err := h.tenantClientChecker.ExistsByTenantAndClient(ctx, tenant.ID, client.ID)
			if err != nil {
				c.Logger().Errorf("logout: failed to check tenant-client: %v", err)
			}
			if !exists {
				c.Logger().Warnf("logout: client %s not linked to tenant %s", client.ClientID, tenantCode)
				hintResult = nil
				client = nil
			}
		}

		if hintResult != nil {
			userSub = hintResult.Subject
			if hintResult.SessionID != "" {
				sessionID, _ = uuid.Parse(hintResult.SessionID)
			}
		}
	}

	// post_logout_redirect_uri 検証
	validPostLogoutRedirectURI := ""
	if postLogoutRedirectURI != "" && client != nil {
		uris, err := h.postLogoutRedirectURILister.ListByClientID(ctx, client.ID)
		if err != nil {
			c.Logger().Errorf("logout: failed to list post_logout_redirect_uris: %v", err)
		} else {
			for _, u := range uris {
				if u.URI == postLogoutRedirectURI {
					validPostLogoutRedirectURI = postLogoutRedirectURI
					break
				}
			}
		}
		if validPostLogoutRedirectURI == "" {
			c.Logger().Warnf("logout: post_logout_redirect_uri not registered: %s", postLogoutRedirectURI)
		}
	}

	// セッション失効（冪等: セッションが見つからなくてもエラーにしない）
	if sessionID != uuid.Nil {
		_ = h.sessionRevoker.Revoke(ctx, sessionID)
		_ = h.accessTokenStore.RevokeBySessionID(ctx, sessionID)
		_ = h.refreshTokenStore.RevokeBySessionID(ctx, sessionID)
	}

	// op_session Cookie も試行的にクリア（id_token_hint がない場合でも）
	if cookie, err := c.Cookie("op_session"); err == nil {
		if cookieSessionID, err := uuid.Parse(cookie.Value); err == nil && cookieSessionID != uuid.Nil {
			// id_token_hint で特定したセッション以外にも Cookie のセッションがある場合は失効
			if cookieSessionID != sessionID {
				_ = h.sessionRevoker.Revoke(ctx, cookieSessionID)
				_ = h.accessTokenStore.RevokeBySessionID(ctx, cookieSessionID)
				_ = h.refreshTokenStore.RevokeBySessionID(ctx, cookieSessionID)
			}
		}
	}
	h.clearSessionCookie(c)

	// SLO 対象クライアント取得
	sloClients, err := h.clientsByTenantLister.ListByTenantIDWithLogoutURIs(ctx, tenant.ID)
	if err != nil {
		c.Logger().Errorf("logout: failed to list SLO clients: %v", err)
		sloClients = nil
	}

	// Back-Channel Logout 通知（非同期、失敗してもブロックしない）
	for i := range sloClients {
		cl := &sloClients[i]
		if cl.BackchannelLogoutURI == nil || *cl.BackchannelLogoutURI == "" {
			continue
		}

		logoutToken, err := h.logoutTokenSigner.SignLogoutToken(ctx, &model.LogoutTokenClaims{
			Issuer:    issuer,
			Subject:   userSub,
			Audience:  cl.ClientID,
			SessionID: sessionID.String(),
		})
		if err != nil {
			c.Logger().Errorf("logout: failed to sign logout_token for client %s: %v", cl.ClientID, err)
			continue
		}

		go sendBackChannelLogout(cl, logoutToken, h.httpClient)
	}

	// Front-Channel Logout 対象 URI 収集
	var frontchannelURIs []string
	for _, cl := range sloClients {
		if cl.FrontchannelLogoutURI != nil && *cl.FrontchannelLogoutURI != "" {
			frontchannelURIs = append(frontchannelURIs, *cl.FrontchannelLogoutURI)
		}
	}

	// リダイレクト先の決定
	if len(frontchannelURIs) > 0 {
		// Front-Channel Logout 対象あり → OP Frontend のログアウトページにリダイレクト
		logoutPageURL, _ := url.Parse(h.frontendBaseURL + "/logout")
		q := logoutPageURL.Query()
		q.Set("frontchannel_uris", strings.Join(frontchannelURIs, ","))
		q.Set("iss", issuer)
		if sessionID != uuid.Nil {
			q.Set("sid", sessionID.String())
		}
		if validPostLogoutRedirectURI != "" {
			q.Set("post_logout_redirect_uri", validPostLogoutRedirectURI)
			if state != "" {
				q.Set("state", state)
			}
		}
		logoutPageURL.RawQuery = q.Encode()
		return c.Redirect(http.StatusFound, logoutPageURL.String())
	}

	if validPostLogoutRedirectURI != "" {
		// Front-Channel なし、post_logout_redirect_uri あり → 直接リダイレクト
		redirectURL, _ := url.Parse(validPostLogoutRedirectURI)
		if state != "" {
			q := redirectURL.Query()
			q.Set("state", state)
			redirectURL.RawQuery = q.Encode()
		}
		return c.Redirect(http.StatusFound, redirectURL.String())
	}

	// どちらもなし → OP Frontend のログアウト完了ページにリダイレクト
	return c.Redirect(http.StatusFound, fmt.Sprintf("%s/logout", h.frontendBaseURL))
}

// clearSessionCookie は op_session Cookie をクリアする。
func (h *LogoutHandler) clearSessionCookie(c echo.Context) {
	cookie := &http.Cookie{
		Name:     "op_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.isSecure,
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)
}
