package oidc

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type DiscoveryHandler struct {
	issuerBaseURL        string
	tenantFinder         TenantFinder
	authDetailTypeFinder AuthorizationDetailTypeFinder
}

func NewDiscoveryHandler(issuerBaseURL string, tenantFinder TenantFinder, authDetailTypeFinder AuthorizationDetailTypeFinder) *DiscoveryHandler {
	return &DiscoveryHandler{
		issuerBaseURL:        issuerBaseURL,
		tenantFinder:         tenantFinder,
		authDetailTypeFinder: authDetailTypeFinder,
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
		"introspection_endpoint":                          issuer + "/introspect",
		"pushed_authorization_request_endpoint":            issuer + "/par",
		"require_pushed_authorization_requests":            false,
		"response_types_supported":              []string{"code"},
		"device_authorization_endpoint":          issuer + "/device/authorize",
		"grant_types_supported":                 []string{"authorization_code", "refresh_token", "client_credentials", "urn:ietf:params:oauth:grant-type:device_code", GrantTypeTokenExchange},
		"subject_types_supported":               []string{"public", "pairwise"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email", "offline_access", "scim"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
		"code_challenge_methods_supported":      []string{"S256"},
		"registration_endpoint":                   issuer + "/register",
		"end_session_endpoint":                    issuer + "/logout",
		"frontchannel_logout_supported":           true,
		"frontchannel_logout_session_supported":   true,
		"backchannel_logout_supported":            true,
		"backchannel_logout_session_supported":    true,
		"acr_values_supported":                  []string{"urn:mace:incommon:iap:bronze"},
		"dpop_signing_alg_values_supported":                []string{"RS256", "ES256"},
		"userinfo_signing_alg_values_supported":            []string{"RS256"},
		"claims_parameter_supported":                       true,
		"claims_supported":                                 []string{"sub", "iss", "aud", "exp", "iat", "auth_time", "nonce", "acr", "amr", "sid", "name", "email", "email_verified", "updated_at"},
	}

	// RFC 9396 Section 9: authorization_details_types_supported
	if h.authDetailTypeFinder != nil {
		adTypes, err := h.authDetailTypeFinder.ListByTenantID(c.Request().Context(), tenant.ID)
		if err == nil && len(adTypes) > 0 {
			typeNames := make([]string, 0, len(adTypes))
			for _, t := range adTypes {
				typeNames = append(typeNames, t.TypeName)
			}
			metadata["authorization_details_types_supported"] = typeNames
		}
	}

	c.Response().Header().Set("Cache-Control", "public, max-age=86400")
	return c.JSON(http.StatusOK, metadata)
}
