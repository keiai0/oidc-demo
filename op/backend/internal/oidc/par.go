package oidc

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// PARHandler は POST /{tenant_code}/par を処理する。
// 仕様参照: RFC 9126
type PARHandler struct {
	clientFinder        ClientFinder
	tenantFinder        TenantFinder
	tenantClientChecker TenantClientChecker
	parStore            PARStore
	verifyPassword      VerifyPasswordFunc
}

// NewPARHandler は PARHandler を生成する。
func NewPARHandler(
	clientFinder ClientFinder,
	tenantFinder TenantFinder,
	tenantClientChecker TenantClientChecker,
	parStore PARStore,
	verifyPassword VerifyPasswordFunc,
) *PARHandler {
	return &PARHandler{
		clientFinder:        clientFinder,
		tenantFinder:        tenantFinder,
		tenantClientChecker: tenantClientChecker,
		parStore:            parStore,
		verifyPassword:      verifyPassword,
	}
}

// Handle は POST /{tenant_code}/par を処理する。
// 仕様参照: RFC 9126 Section 2
func (h *PARHandler) Handle(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Pragma", "no-cache")

	ctx := c.Request().Context()
	tenantCode := c.Param("tenant_code")

	// テナント検証
	tenant, err := h.tenantFinder.FindByCode(ctx, tenantCode)
	if err != nil {
		return tokenError(c, http.StatusInternalServerError, "server_error", "")
	}
	if tenant == nil {
		return tokenError(c, http.StatusBadRequest, "invalid_request", "unknown tenant")
	}

	// クライアント認証 (MUST: RFC 9126 Section 2)
	clientID, clientSecret := extractClientCredentials(c)
	if clientID == "" || clientSecret == "" {
		return tokenError(c, http.StatusUnauthorized, "invalid_client", "client credentials required")
	}

	client, err := h.clientFinder.FindByClientIDWithRedirectURIs(ctx, clientID)
	if err != nil {
		return tokenError(c, http.StatusInternalServerError, "server_error", "")
	}
	if client == nil || client.Status != "active" {
		return tokenError(c, http.StatusUnauthorized, "invalid_client", "")
	}

	match, err := h.verifyPassword(clientSecret, client.ClientSecretHash)
	if err != nil || !match {
		return tokenError(c, http.StatusUnauthorized, "invalid_client", "")
	}

	// テナント-クライアント紐づきチェック
	belongs, err := h.tenantClientChecker.ExistsByTenantAndClient(ctx, tenant.ID, client.ID)
	if err != nil {
		return tokenError(c, http.StatusInternalServerError, "server_error", "")
	}
	if !belongs {
		return tokenError(c, http.StatusBadRequest, "invalid_request", "client does not belong to this tenant")
	}

	// 認可リクエストパラメータの検証
	responseType := c.FormValue("response_type")
	redirectURI := c.FormValue("redirect_uri")
	scope := c.FormValue("scope")
	codeChallenge := c.FormValue("code_challenge")
	codeChallengeMethod := c.FormValue("code_challenge_method")

	if responseType != "code" {
		return tokenError(c, http.StatusBadRequest, "invalid_request", "only response_type=code is supported")
	}

	if redirectURI == "" {
		return tokenError(c, http.StatusBadRequest, "invalid_request", "redirect_uri is required")
	}
	if !isRegisteredRedirectURI(client.RedirectURIs, redirectURI) {
		return tokenError(c, http.StatusBadRequest, "invalid_request", "redirect_uri mismatch")
	}

	scopes := strings.Split(scope, " ")
	if !containsScope(scopes, "openid") {
		return tokenError(c, http.StatusBadRequest, "invalid_request", "openid scope is required")
	}

	if client.RequirePKCE {
		if codeChallenge == "" {
			return tokenError(c, http.StatusBadRequest, "invalid_request", "code_challenge is required")
		}
		if codeChallengeMethod != "S256" {
			return tokenError(c, http.StatusBadRequest, "invalid_request", "only S256 code_challenge_method is supported")
		}
	}

	// パラメータを JSON に変換して保存
	params := map[string]string{
		"response_type":         responseType,
		"client_id":             clientID,
		"redirect_uri":          redirectURI,
		"scope":                 scope,
		"state":                 c.FormValue("state"),
		"nonce":                 c.FormValue("nonce"),
		"code_challenge":        codeChallenge,
		"code_challenge_method": codeChallengeMethod,
		"prompt":                c.FormValue("prompt"),
		"max_age":               c.FormValue("max_age"),
		"acr_values":            c.FormValue("acr_values"),
		"claims":                c.FormValue("claims"),
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return tokenError(c, http.StatusInternalServerError, "server_error", "")
	}

	// request_uri 生成 (RFC 9126 Section 2.2)
	requestURI := "urn:ietf:params:oauth:request_uri:" + uuid.New().String()

	par := &model.PushedAuthorizationRequest{
		RequestURI: requestURI,
		ClientID:   client.ID,
		Parameters: paramsJSON,
		ExpiresAt:  time.Now().Add(60 * time.Second),
	}

	if err := h.parStore.Create(ctx, par); err != nil {
		return tokenError(c, http.StatusInternalServerError, "server_error", "")
	}

	// RFC 9126 Section 2.2: 201 Created
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"request_uri": requestURI,
		"expires_in":  60,
	})
}
