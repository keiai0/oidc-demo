package oidc

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/auth"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// DemoAuthCodeHandler はデモモード専用の認可コード発行ハンドラ。
// 認証→同意→コード発行を 1 リクエストで実行する（CSRF 攻撃デモ用）。
type DemoAuthCodeHandler struct {
	authSvc             *auth.AuthService
	clientFinder        ClientFinder
	tenantFinder        TenantFinder
	tenantClientChecker TenantClientChecker
	authCodeStore       AuthorizationCodeStore
}

// NewDemoAuthCodeHandler は DemoAuthCodeHandler を生成する。
func NewDemoAuthCodeHandler(
	authSvc *auth.AuthService,
	clientFinder ClientFinder,
	tenantFinder TenantFinder,
	tenantClientChecker TenantClientChecker,
	authCodeStore AuthorizationCodeStore,
) *DemoAuthCodeHandler {
	return &DemoAuthCodeHandler{
		authSvc:             authSvc,
		clientFinder:        clientFinder,
		tenantFinder:        tenantFinder,
		tenantClientChecker: tenantClientChecker,
		authCodeStore:       authCodeStore,
	}
}

type demoAuthCodeRequest struct {
	LoginID     string `json:"login_id"`
	Password    string `json:"password"`
	ClientID    string `json:"client_id"`
	RedirectURI string `json:"redirect_uri"`
	Scope       string `json:"scope"`
}

// Handle は POST /api/demo/auth-code を処理する。
func (h *DemoAuthCodeHandler) Handle(c echo.Context) error {
	var req demoAuthCodeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
	}

	if req.LoginID == "" || req.Password == "" || req.ClientID == "" || req.RedirectURI == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":             "invalid_request",
			"error_description": "login_id, password, client_id, and redirect_uri are required",
		})
	}

	ctx := c.Request().Context()

	// 1. ユーザー認証（demo テナント固定）
	loginInput := &model.LoginInput{
		TenantCode: "demo",
		LoginID:    req.LoginID,
		Password:   req.Password,
		IPAddress:  c.RealIP(),
		UserAgent:  "demo-csrf-attack",
	}
	loginOutput, err := h.authSvc.Login(ctx, loginInput)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication_failed"})
	}

	// 2. クライアント検索 + redirect_uri 検証
	client, err := h.clientFinder.FindByClientIDWithRedirectURIs(ctx, req.ClientID)
	if err != nil || client == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_client"})
	}

	// redirect_uri がクライアントに登録されているか確認
	redirectURIValid := false
	for _, uri := range client.RedirectURIs {
		if uri.URI == req.RedirectURI {
			redirectURIValid = true
			break
		}
	}
	if !redirectURIValid {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_redirect_uri"})
	}

	// テナント-クライアント紐づけ確認
	tenant, err := h.tenantFinder.FindByCode(ctx, "demo")
	if err != nil || tenant == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	exists, err := h.tenantClientChecker.ExistsByTenantAndClient(ctx, tenant.ID, client.ID)
	if err != nil || !exists {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_client"})
	}

	// 3. 認可コード生成
	codeBytes := make([]byte, 32)
	if _, err := rand.Read(codeBytes); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	code := hex.EncodeToString(codeBytes)

	scope := req.Scope
	if scope == "" {
		scope = "openid"
	}

	authCode := &model.AuthorizationCode{
		SessionID:   loginOutput.SessionID,
		ClientID:    client.ID,
		Code:        code,
		RedirectURI: req.RedirectURI,
		Scope:       scope,
		ExpiresAt:   time.Now().Add(time.Duration(tenant.AuthCodeLifetime) * time.Second),
	}

	if err := h.authCodeStore.Create(ctx, authCode); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	// 4. callback URL を組み立て（state は付与しない — CSRF デモのため）
	callbackURL, err := url.Parse(req.RedirectURI)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	q := callbackURL.Query()
	q.Set("code", code)
	callbackURL.RawQuery = q.Encode()

	return c.JSON(http.StatusOK, map[string]string{
		"callback_url": callbackURL.String(),
	})
}
