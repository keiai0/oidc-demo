package oidc

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/audit"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/metrics"
)

type TokenHandler struct {
	authCodeStore       AuthorizationCodeStore
	accessTokenStore    AccessTokenStore
	refreshTokenStore   RefreshTokenStore
	idTokenCreator      IDTokenCreator
	clientFinder        ClientFinder
	tenantFinder        TenantFinder
	tenantClientChecker TenantClientChecker
	userFinder          UserFinder
	tokenSigner         TokenSigner
	verifyPassword      VerifyPasswordFunc
	verifyCodeChallenge VerifyCodeChallengeFunc
	computeATHash       ComputeATHashFunc
	sha256Hex           SHA256HexFunc
	dpopJTIStore        DPoPJTIStore
	deviceAuthStore     DeviceAuthorizationRequestStore
	sessionFinder       SessionFinder
	audit               *audit.AuditLogger
	logger              *slog.Logger
	issuerBaseURL       string
	demoMode            bool
}

func NewTokenHandler(
	authCodeStore AuthorizationCodeStore,
	accessTokenStore AccessTokenStore,
	refreshTokenStore RefreshTokenStore,
	idTokenCreator IDTokenCreator,
	clientFinder ClientFinder,
	tenantFinder TenantFinder,
	tenantClientChecker TenantClientChecker,
	userFinder UserFinder,
	tokenSigner TokenSigner,
	verifyPassword VerifyPasswordFunc,
	verifyCodeChallenge VerifyCodeChallengeFunc,
	computeATHash ComputeATHashFunc,
	sha256Hex SHA256HexFunc,
	dpopJTIStore DPoPJTIStore,
	deviceAuthStore DeviceAuthorizationRequestStore,
	sessionFinder SessionFinder,
	auditLog *audit.AuditLogger,
	logger *slog.Logger,
	issuerBaseURL string,
	demoMode bool,
) *TokenHandler {
	return &TokenHandler{
		authCodeStore:       authCodeStore,
		accessTokenStore:    accessTokenStore,
		refreshTokenStore:   refreshTokenStore,
		idTokenCreator:      idTokenCreator,
		clientFinder:        clientFinder,
		tenantFinder:        tenantFinder,
		tenantClientChecker: tenantClientChecker,
		userFinder:          userFinder,
		tokenSigner:         tokenSigner,
		verifyPassword:      verifyPassword,
		verifyCodeChallenge: verifyCodeChallenge,
		computeATHash:       computeATHash,
		sha256Hex:           sha256Hex,
		dpopJTIStore:        dpopJTIStore,
		deviceAuthStore:     deviceAuthStore,
		sessionFinder:       sessionFinder,
		audit:               auditLog,
		logger:              logger,
		issuerBaseURL:       issuerBaseURL,
		demoMode:            demoMode,
	}
}

// Handle は POST /{tenant_code}/token を処理する
// 仕様参照: RFC 6749 Section 4.1.3, OIDC Core 1.0 Section 3.1.3
func (h *TokenHandler) Handle(c echo.Context) error {
	// Cache-Control: no-store (MUST: RFC 6749 Section 5.1)
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Pragma", "no-cache")

	// DPoP proof 検証 (RFC 9449)
	var dpopJKT string
	dpopProof := c.Request().Header.Get("DPoP")
	if dpopProof != "" {
		// htu はクライアントが知る外部 URL で検証する（Docker 内部 URL ではなく）
		tenantCode := c.Param("tenant_code")
		httpURL := h.issuerBaseURL + "/" + tenantCode + "/token"

		result, err := VerifyDPoPProof(c.Request().Context(), dpopProof, "POST", httpURL, h.dpopJTIStore)
		if err != nil {
			return tokenError(c, http.StatusBadRequest, "invalid_dpop_proof", err.Error())
		}
		dpopJKT = result.JKT
	}

	grantType := c.FormValue("grant_type")

	switch grantType {
	case "authorization_code":
		return h.handleAuthCodeGrant(c, dpopJKT)
	case "refresh_token":
		return h.handleRefreshTokenGrant(c, dpopJKT)
	case "client_credentials":
		return h.handleClientCredentialsGrant(c, dpopJKT)
	case "urn:ietf:params:oauth:grant-type:device_code":
		return h.handleDeviceCodeGrant(c, dpopJKT)
	default:
		return tokenError(c, http.StatusBadRequest, "unsupported_grant_type", "")
	}
}

func (h *TokenHandler) handleAuthCodeGrant(c echo.Context, dpopJKT string) error {
	// クライアント認証: client_secret_post または client_secret_basic
	clientID, clientSecret := extractClientCredentials(c)
	if clientID == "" || clientSecret == "" {
		return tokenError(c, http.StatusUnauthorized, "invalid_client", "client credentials required")
	}

	code := c.FormValue("code")
	redirectURI := c.FormValue("redirect_uri")
	codeVerifier := c.FormValue("code_verifier")

	if code == "" {
		return tokenError(c, http.StatusBadRequest, "invalid_request", "code is required")
	}

	resp, err := h.handleAuthCodeGrantLogic(c.Request().Context(), &AuthCodeGrantInput{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Code:         code,
		RedirectURI:  redirectURI,
		CodeVerifier: codeVerifier,
		DPoPJKT:      dpopJKT,
	})
	if err != nil {
		if errors.Is(err, ErrInvalidClient) {
			return tokenError(c, http.StatusUnauthorized, "invalid_client", "")
		}
		if errors.Is(err, ErrInvalidGrant) {
			return tokenError(c, http.StatusBadRequest, "invalid_grant", "")
		}
		h.logger.ErrorContext(c.Request().Context(), "token error", "error", err)
		return tokenError(c, http.StatusInternalServerError, "server_error", "")
	}

	h.audit.LogEvent(c.Request().Context(), audit.EventTokenIssued,
		audit.ClientAttr(clientID), audit.GrantTypeAttr("authorization_code"), audit.ResultAttr("success"),
	)
	metrics.TokenIssuedTotal.WithLabelValues("authorization_code").Inc()

	return c.JSON(http.StatusOK, resp)
}

func (h *TokenHandler) handleRefreshTokenGrant(c echo.Context, dpopJKT string) error {
	clientID, clientSecret := extractClientCredentials(c)
	if clientID == "" || clientSecret == "" {
		return tokenError(c, http.StatusUnauthorized, "invalid_client", "client credentials required")
	}

	refreshToken := c.FormValue("refresh_token")
	if refreshToken == "" {
		return tokenError(c, http.StatusBadRequest, "invalid_request", "refresh_token is required")
	}

	scope := c.FormValue("scope")

	resp, err := h.handleRefreshTokenGrantLogic(c.Request().Context(), &RefreshTokenGrantInput{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RefreshToken: refreshToken,
		Scope:        scope,
		DPoPJKT:      dpopJKT,
	})
	if err != nil {
		if errors.Is(err, ErrInvalidClient) {
			return tokenError(c, http.StatusUnauthorized, "invalid_client", "")
		}
		if errors.Is(err, ErrInvalidGrant) {
			return tokenError(c, http.StatusBadRequest, "invalid_grant", "")
		}
		if errors.Is(err, ErrUnsupportedGrantType) {
			return tokenError(c, http.StatusBadRequest, "unsupported_grant_type", "")
		}
		h.logger.ErrorContext(c.Request().Context(), "refresh token error", "error", err)
		return tokenError(c, http.StatusInternalServerError, "server_error", "")
	}

	h.audit.LogEvent(c.Request().Context(), audit.EventTokenIssued,
		audit.ClientAttr(clientID), audit.GrantTypeAttr("refresh_token"), audit.ResultAttr("success"),
	)
	metrics.TokenIssuedTotal.WithLabelValues("refresh_token").Inc()

	return c.JSON(http.StatusOK, resp)
}

// extractClientCredentials は client_secret_post と client_secret_basic の両方をサポートする
func extractClientCredentials(c echo.Context) (clientID, clientSecret string) {
	// client_secret_post
	clientID = c.FormValue("client_id")
	clientSecret = c.FormValue("client_secret")
	if clientID != "" && clientSecret != "" {
		return
	}

	// client_secret_basic (Authorization: Basic base64(client_id:client_secret))
	clientID, clientSecret, ok := c.Request().BasicAuth()
	if ok && clientID != "" && clientSecret != "" {
		return
	}

	return "", ""
}

func tokenError(c echo.Context, status int, errCode, errDescription string) error {
	body := map[string]string{"error": errCode}
	if errDescription != "" {
		body["error_description"] = errDescription
	}
	return c.JSON(status, body)
}
