package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/audit"
)

// FederationHandler は外部 IdP 連携の Internal API を処理する。
type FederationHandler struct {
	federationSvc *FederationService
	audit         *audit.AuditLogger
	frontendURL   string
	backendURL    string
	secure        bool
}

// NewFederationHandler は FederationHandler を生成する。
func NewFederationHandler(
	federationSvc *FederationService,
	auditLog *audit.AuditLogger,
	frontendURL, backendURL string,
	secure bool,
) *FederationHandler {
	return &FederationHandler{
		federationSvc: federationSvc,
		audit:         auditLog,
		frontendURL:   frontendURL,
		backendURL:    backendURL,
		secure:        secure,
	}
}

type federationProviderListItem struct {
	Name string `json:"name"`
}

// HandleListProviders は GET /internal/federation/providers?tenant_code=xxx を処理する。
// テナントのアクティブな連携プロバイダ一覧を返す。
func (h *FederationHandler) HandleListProviders(c echo.Context) error {
	tenantCode := c.QueryParam("tenant_code")
	if tenantCode == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_code is required"})
	}

	providers, err := h.federationSvc.ListProviders(c.Request().Context(), tenantCode)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	items := make([]federationProviderListItem, len(providers))
	for i, p := range providers {
		items[i] = federationProviderListItem{Name: p.Name}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"providers": items})
}

// HandleInitiate は GET /internal/federation/:provider/initiate を処理する。
// 外部 IdP への認可リクエスト URL を生成し、リダイレクトレスポンスを返す。
func (h *FederationHandler) HandleInitiate(c echo.Context) error {
	providerName := c.Param("provider")
	tenantCode := c.QueryParam("tenant_code")
	redirectAfterLogin := c.QueryParam("redirect_after_login")

	if tenantCode == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_code is required"})
	}

	// state と nonce を生成
	state, err := generateRandomString(32)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	nonce, err := generateRandomString(32)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	// コールバック URL
	callbackURL := h.backendURL + "/internal/federation/" + providerName + "/callback"

	result, err := h.federationSvc.InitiateFederation(
		c.Request().Context(),
		tenantCode, providerName, state, nonce, callbackURL,
	)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// state をセッション Cookie に保存（CSRF 防止）
	c.SetCookie(&http.Cookie{
		Name:     "federation_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})
	// テナントコードとリダイレクト先も保存
	c.SetCookie(&http.Cookie{
		Name:     "federation_tenant",
		Value:    tenantCode,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})
	if redirectAfterLogin != "" {
		c.SetCookie(&http.Cookie{
			Name:     "federation_redirect",
			Value:    redirectAfterLogin,
			Path:     "/",
			HttpOnly: true,
			Secure:   h.secure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   600,
		})
	}

	return c.Redirect(http.StatusFound, result.AuthorizationURL)
}

// HandleCallback は GET /internal/federation/:provider/callback を処理する。
// 外部 IdP からのコールバックを受け、セッションを作成しフロントエンドにリダイレクトする。
func (h *FederationHandler) HandleCallback(c echo.Context) error {
	providerName := c.Param("provider")
	code := c.QueryParam("code")
	state := c.QueryParam("state")

	if code == "" {
		errorParam := c.QueryParam("error")
		errorDesc := c.QueryParam("error_description")
		return c.Redirect(http.StatusFound, h.frontendURL+"/login?error=federation_failed&error_description="+errorDesc+"&provider="+providerName+"&external_error="+errorParam)
	}

	// state 検証（CSRF 防止）
	stateCookie, err := c.Cookie("federation_state")
	if err != nil || stateCookie.Value != state {
		return c.Redirect(http.StatusFound, h.frontendURL+"/login?error=invalid_state")
	}

	// テナントコード取得
	tenantCookie, err := c.Cookie("federation_tenant")
	if err != nil {
		return c.Redirect(http.StatusFound, h.frontendURL+"/login?error=missing_tenant")
	}
	tenantCode := tenantCookie.Value

	// コールバック URL（token exchange 用）
	callbackURL := h.backendURL + "/internal/federation/" + providerName + "/callback"

	result, err := h.federationSvc.HandleCallback(
		c.Request().Context(),
		tenantCode, providerName, code, callbackURL,
		c.RealIP(), c.Request().UserAgent(),
	)
	if err != nil {
		return c.Redirect(http.StatusFound, h.frontendURL+"/login?error=federation_failed&error_description="+err.Error())
	}

	// セッション Cookie を設定
	c.SetCookie(&http.Cookie{
		Name:     "op_session",
		Value:    result.Session.ID.String(),
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
	})

	// federation 用の一時 Cookie をクリア
	for _, name := range []string{"federation_state", "federation_tenant", "federation_redirect"} {
		c.SetCookie(&http.Cookie{
			Name:   name,
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
	}

	h.audit.LogEvent(c.Request().Context(), audit.EventFederationLogin,
		audit.UserAttr(result.Session.UserID.String()),
		audit.TenantAttr(tenantCode),
	)

	// リダイレクト先
	redirectCookie, err := c.Cookie("federation_redirect")
	if err == nil && redirectCookie.Value != "" {
		return c.Redirect(http.StatusFound, redirectCookie.Value)
	}

	return c.Redirect(http.StatusFound, h.frontendURL+"/")
}

func generateRandomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
