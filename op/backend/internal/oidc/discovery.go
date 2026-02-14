package oidc

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type DiscoveryHandler struct {
	issuerBaseURL string
	tenantFinder  TenantFinder
}

func NewDiscoveryHandler(issuerBaseURL string, tenantFinder TenantFinder) *DiscoveryHandler {
	return &DiscoveryHandler{
		issuerBaseURL: issuerBaseURL,
		tenantFinder:  tenantFinder,
	}
}

// Handle は GET /{tenant_code}/.well-known/openid-configuration を処理する
// 仕様参照: OIDC Discovery 1.0 Section 3, 4
func (h *DiscoveryHandler) Handle(c echo.Context) error {
	tenantCode := c.Param("tenant_code")

	tenant, err := h.tenantFinder.FindByCode(c.Request().Context(), tenantCode)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	if tenant == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not_found"})
	}

	// issuer は末尾スラッシュを含めない (MUST: OIDC Discovery 1.0 Section 4.1)
	issuer := h.issuerBaseURL + "/" + tenantCode

	metadata := map[string]interface{}{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/authorize",
		"token_endpoint":                        issuer + "/token",
		"userinfo_endpoint":                     issuer + "/userinfo",
		"jwks_uri":                              h.issuerBaseURL + "/jwks",
		"revocation_endpoint":                   issuer + "/revoke",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email", "offline_access"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
		"code_challenge_methods_supported":      []string{"S256"},
		"claims_supported":                      []string{"sub", "iss", "aud", "exp", "iat", "auth_time", "nonce", "name", "email", "email_verified"},
	}

	c.Response().Header().Set("Cache-Control", "public, max-age=86400")
	return c.JSON(http.StatusOK, metadata)
}
