package oidc

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// サポートする grant_types / response_types / auth_methods
var (
	supportedGrantTypes = map[string]bool{
		"authorization_code": true,
		"refresh_token":      true,
		"client_credentials": true,
	}
	supportedResponseTypes = map[string]bool{
		"code": true,
	}
	supportedAuthMethods = map[string]bool{
		"client_secret_basic": true,
		"client_secret_post":  true,
		"none":                true,
	}
)

// RegistrationHandler は Dynamic Client Registration (RFC 7591/7592) のエンドポイントを処理する。
type RegistrationHandler struct {
	tenantFinder    TenantFinder
	clientFinder    ClientFinder
	clientCreator   ClientCreator
	clientUpdater   ClientUpdater
	tenantClientChecker TenantClientChecker
	tenantClientCreator TenantClientCreator
	redirectURICreator  RedirectURICreator
	iatFinder       InitialAccessTokenFinder
	regStore        ClientRegistrationStore
	hashPassword    HashPasswordFunc
	sha256Hex       SHA256HexFunc
	baseURL         string
}

// NewRegistrationHandler は RegistrationHandler を生成する。
func NewRegistrationHandler(
	tenantFinder TenantFinder,
	clientFinder ClientFinder,
	clientCreator ClientCreator,
	clientUpdater ClientUpdater,
	tenantClientChecker TenantClientChecker,
	tenantClientCreator TenantClientCreator,
	redirectURICreator RedirectURICreator,
	iatFinder InitialAccessTokenFinder,
	regStore ClientRegistrationStore,
	hashPassword HashPasswordFunc,
	sha256Hex SHA256HexFunc,
	baseURL string,
) *RegistrationHandler {
	return &RegistrationHandler{
		tenantFinder:        tenantFinder,
		clientFinder:        clientFinder,
		clientCreator:       clientCreator,
		clientUpdater:       clientUpdater,
		tenantClientChecker: tenantClientChecker,
		tenantClientCreator: tenantClientCreator,
		redirectURICreator:  redirectURICreator,
		iatFinder:           iatFinder,
		regStore:            regStore,
		hashPassword:        hashPassword,
		sha256Hex:           sha256Hex,
		baseURL:             baseURL,
	}
}

// registrationRequest は RFC 7591 Section 3.1 のクライアント登録リクエスト。
type registrationRequest struct {
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	ClientName              string   `json:"client_name,omitempty"`
	ClientURI               string   `json:"client_uri,omitempty"`
	LogoURI                 string   `json:"logo_uri,omitempty"`
	Scope                   string   `json:"scope,omitempty"`
	Contacts                []string `json:"contacts,omitempty"`
	TosURI                  string   `json:"tos_uri,omitempty"`
	PolicyURI               string   `json:"policy_uri,omitempty"`
	SoftwareID              string   `json:"software_id,omitempty"`
	SoftwareVersion         string   `json:"software_version,omitempty"`
	SoftwareStatement       string   `json:"software_statement,omitempty"`
}

// registrationResponse は RFC 7591 Section 3.2 のクライアント登録レスポンス。
type registrationResponse struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret,omitempty"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	ClientSecretExpiresAt   int64    `json:"client_secret_expires_at"`
	RegistrationAccessToken string   `json:"registration_access_token"`
	RegistrationClientURI   string   `json:"registration_client_uri"`
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	ClientName              string   `json:"client_name,omitempty"`
	SoftwareID              string   `json:"software_id,omitempty"`
	SoftwareVersion         string   `json:"software_version,omitempty"`
}

// clientConfigResponse は RFC 7592 のクライアント設定レスポンス（secret を含まない）。
type clientConfigResponse struct {
	ClientID                string   `json:"client_id"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	RegistrationAccessToken string   `json:"registration_access_token,omitempty"`
	RegistrationClientURI   string   `json:"registration_client_uri"`
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	ClientName              string   `json:"client_name,omitempty"`
	SoftwareID              string   `json:"software_id,omitempty"`
	SoftwareVersion         string   `json:"software_version,omitempty"`
}

// HandleRegister は POST /{tenant_code}/register を処理する (RFC 7591 Section 3.1)。
func (h *RegistrationHandler) HandleRegister(c echo.Context) error {
	ctx := c.Request().Context()
	tenantCode := c.Param("tenant_code")

	// テナント検証
	tenant, err := h.tenantFinder.FindByCode(ctx, tenantCode)
	if err != nil {
		return registrationError(c, http.StatusInternalServerError, "server_error", "")
	}
	if tenant == nil {
		return registrationError(c, http.StatusNotFound, "invalid_client_metadata", "unknown tenant")
	}

	// Initial Access Token の検証 (RFC 7591 Section 1.2)
	bearerToken := extractBearerToken(c)
	if bearerToken == "" {
		return registrationError(c, http.StatusUnauthorized, "invalid_token", "initial access token required")
	}

	tokenHash := h.sha256Hex(bearerToken)
	iat, err := h.iatFinder.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return registrationError(c, http.StatusInternalServerError, "server_error", "")
	}
	if iat == nil {
		return registrationError(c, http.StatusUnauthorized, "invalid_token", "invalid initial access token")
	}
	if !iat.IsValid() {
		if iat.IsExpired() {
			return registrationError(c, http.StatusUnauthorized, "invalid_token", "initial access token expired")
		}
		if iat.IsRevoked() {
			return registrationError(c, http.StatusUnauthorized, "invalid_token", "initial access token revoked")
		}
		if iat.IsExhausted() {
			return registrationError(c, http.StatusForbidden, "invalid_token", "initial access token usage limit exceeded")
		}
	}
	// IAT のテナントが一致するか
	if iat.TenantID != tenant.ID {
		return registrationError(c, http.StatusForbidden, "invalid_token", "token not valid for this tenant")
	}

	// リクエストボディのパース
	var req registrationRequest
	if err := c.Bind(&req); err != nil {
		return registrationError(c, http.StatusBadRequest, "invalid_client_metadata", "invalid request body")
	}

	// メタデータバリデーション
	if err := h.validateRegistrationRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	// デフォルト値の適用 (RFC 7591 Section 2)
	if req.TokenEndpointAuthMethod == "" {
		req.TokenEndpointAuthMethod = "client_secret_basic"
	}
	if len(req.GrantTypes) == 0 {
		req.GrantTypes = []string{"authorization_code"}
	}
	if len(req.ResponseTypes) == 0 {
		req.ResponseTypes = []string{"code"}
	}

	// クライアント名のデフォルト
	clientName := req.ClientName
	if clientName == "" {
		clientName = "Dynamic Client"
	}

	// client_id / client_secret 生成
	clientID, err := generateRandomHex(16)
	if err != nil {
		return registrationError(c, http.StatusInternalServerError, "server_error", "")
	}

	var clientSecret string
	var secretHash string
	// public client (token_endpoint_auth_method=none) にはシークレットを発行しない
	if req.TokenEndpointAuthMethod != "none" {
		clientSecret, err = generateRandomHex(32)
		if err != nil {
			return registrationError(c, http.StatusInternalServerError, "server_error", "")
		}
		secretHash, err = h.hashPassword(clientSecret)
		if err != nil {
			return registrationError(c, http.StatusInternalServerError, "server_error", "")
		}
	}

	// Registration Access Token 生成
	regAccessToken, err := generateRandomHex(32)
	if err != nil {
		return registrationError(c, http.StatusInternalServerError, "server_error", "")
	}
	regAccessTokenHash := h.sha256Hex(regAccessToken)

	// registration_client_uri の構築
	regClientURI := fmt.Sprintf("%s/%s/register/%s", h.baseURL, tenantCode, clientID)

	// クライアント作成
	client := &model.Client{
		ClientID:                clientID,
		ClientSecretHash:        secretHash,
		Name:                    clientName,
		GrantTypes:              model.StringSlice(req.GrantTypes),
		ResponseTypes:           model.StringSlice(req.ResponseTypes),
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		RequirePKCE:             true,
		Status:                  "active",
		SubjectType:             "public",
	}

	// Redirect URI を関連として設定
	for _, uri := range req.RedirectURIs {
		client.RedirectURIs = append(client.RedirectURIs, model.RedirectURI{URI: uri})
	}

	if err := h.clientCreator.Create(ctx, client); err != nil {
		return registrationError(c, http.StatusInternalServerError, "server_error", "")
	}

	// テナント-クライアント紐づけ
	tc := &model.TenantClient{
		TenantID: tenant.ID,
		ClientID: client.ID,
		Enabled:  true,
	}
	if err := h.tenantClientCreator.Create(ctx, tc); err != nil {
		return registrationError(c, http.StatusInternalServerError, "server_error", "")
	}

	// 登録メタデータ保存
	var softwareID *string
	var softwareVersion *string
	if req.SoftwareID != "" {
		softwareID = &req.SoftwareID
	}
	if req.SoftwareVersion != "" {
		softwareVersion = &req.SoftwareVersion
	}

	reg := &model.ClientRegistration{
		ClientID:                    client.ID,
		RegistrationAccessTokenHash: regAccessTokenHash,
		RegistrationClientURI:       regClientURI,
		SoftwareID:                  softwareID,
		SoftwareVersion:             softwareVersion,
		InitialAccessTokenID:        &iat.ID,
	}
	if err := h.regStore.Create(ctx, reg); err != nil {
		return registrationError(c, http.StatusInternalServerError, "server_error", "")
	}

	// IAT の使用回数をインクリメント
	if err := h.iatFinder.IncrementUsedCount(ctx, iat.ID); err != nil {
		// ログのみ（登録自体は成功させる）
		c.Logger().Errorf("failed to increment IAT used_count: %v", err)
	}

	// レスポンス (RFC 7591 Section 3.2.1)
	resp := registrationResponse{
		ClientID:                clientID,
		ClientSecret:            clientSecret,
		ClientIDIssuedAt:        time.Now().Unix(),
		ClientSecretExpiresAt:   0, // 無期限
		RegistrationAccessToken: regAccessToken,
		RegistrationClientURI:   regClientURI,
		RedirectURIs:            req.RedirectURIs,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		GrantTypes:              req.GrantTypes,
		ResponseTypes:           req.ResponseTypes,
		ClientName:              clientName,
		SoftwareID:              req.SoftwareID,
		SoftwareVersion:         req.SoftwareVersion,
	}

	return c.JSON(http.StatusCreated, resp)
}

// HandleGetClient は GET /{tenant_code}/register/{client_id} を処理する (RFC 7592 Section 2.1)。
func (h *RegistrationHandler) HandleGetClient(c echo.Context) error {
	clientID := c.Param("client_id")

	// Registration Access Token 認証
	reg, client, err := h.authenticateRegistrationRequest(c, clientID)
	if err != nil {
		return err // already wrote response
	}

	redirectURIs := make([]string, len(client.RedirectURIs))
	for i, ru := range client.RedirectURIs {
		redirectURIs[i] = ru.URI
	}

	resp := clientConfigResponse{
		ClientID:                client.ClientID,
		ClientIDIssuedAt:        client.CreatedAt.Unix(),
		RegistrationClientURI:   reg.RegistrationClientURI,
		RedirectURIs:            redirectURIs,
		TokenEndpointAuthMethod: client.TokenEndpointAuthMethod,
		GrantTypes:              []string(client.GrantTypes),
		ResponseTypes:           []string(client.ResponseTypes),
		ClientName:              client.Name,
	}
	if reg.SoftwareID != nil {
		resp.SoftwareID = *reg.SoftwareID
	}
	if reg.SoftwareVersion != nil {
		resp.SoftwareVersion = *reg.SoftwareVersion
	}

	return c.JSON(http.StatusOK, resp)
}

// HandleUpdateClient は PUT /{tenant_code}/register/{client_id} を処理する (RFC 7592 Section 2.2)。
func (h *RegistrationHandler) HandleUpdateClient(c echo.Context) error {
	ctx := c.Request().Context()
	clientIDParam := c.Param("client_id")

	// Registration Access Token 認証
	reg, client, err := h.authenticateRegistrationRequest(c, clientIDParam)
	if err != nil {
		return err
	}

	var req registrationRequest
	if err := c.Bind(&req); err != nil {
		return registrationError(c, http.StatusBadRequest, "invalid_client_metadata", "invalid request body")
	}

	// バリデーション
	if err := h.validateRegistrationRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	// デフォルト値
	if req.TokenEndpointAuthMethod == "" {
		req.TokenEndpointAuthMethod = "client_secret_basic"
	}
	if len(req.GrantTypes) == 0 {
		req.GrantTypes = []string{"authorization_code"}
	}
	if len(req.ResponseTypes) == 0 {
		req.ResponseTypes = []string{"code"}
	}

	clientName := req.ClientName
	if clientName == "" {
		clientName = client.Name
	}

	// クライアント更新（全フィールド置き換え: RFC 7592 Section 2.2）
	client.Name = clientName
	client.GrantTypes = model.StringSlice(req.GrantTypes)
	client.ResponseTypes = model.StringSlice(req.ResponseTypes)
	client.TokenEndpointAuthMethod = req.TokenEndpointAuthMethod

	if err := h.clientUpdater.Update(ctx, client); err != nil {
		return registrationError(c, http.StatusInternalServerError, "server_error", "")
	}

	// Redirect URI の更新（全置き換え）
	if err := h.redirectURICreator.DeleteByClientID(ctx, client.ID); err != nil {
		return registrationError(c, http.StatusInternalServerError, "server_error", "")
	}
	for _, uri := range req.RedirectURIs {
		ru := &model.RedirectURI{ClientDBID: client.ID, URI: uri}
		if err := h.redirectURICreator.Create(ctx, ru); err != nil {
			return registrationError(c, http.StatusInternalServerError, "server_error", "")
		}
	}

	// Registration Access Token のローテーション (RFC 7592 Section 2.2)
	newRegToken, err := generateRandomHex(32)
	if err != nil {
		return registrationError(c, http.StatusInternalServerError, "server_error", "")
	}
	reg.RegistrationAccessTokenHash = h.sha256Hex(newRegToken)

	// software metadata の更新
	if req.SoftwareID != "" {
		reg.SoftwareID = &req.SoftwareID
	}
	if req.SoftwareVersion != "" {
		reg.SoftwareVersion = &req.SoftwareVersion
	}

	if err := h.regStore.Update(ctx, reg); err != nil {
		return registrationError(c, http.StatusInternalServerError, "server_error", "")
	}

	resp := clientConfigResponse{
		ClientID:                client.ClientID,
		ClientIDIssuedAt:        client.CreatedAt.Unix(),
		RegistrationAccessToken: newRegToken,
		RegistrationClientURI:   reg.RegistrationClientURI,
		RedirectURIs:            req.RedirectURIs,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		GrantTypes:              req.GrantTypes,
		ResponseTypes:           req.ResponseTypes,
		ClientName:              clientName,
	}
	if reg.SoftwareID != nil {
		resp.SoftwareID = *reg.SoftwareID
	}
	if reg.SoftwareVersion != nil {
		resp.SoftwareVersion = *reg.SoftwareVersion
	}

	return c.JSON(http.StatusOK, resp)
}

// HandleDeleteClient は DELETE /{tenant_code}/register/{client_id} を処理する (RFC 7592 Section 2.3)。
func (h *RegistrationHandler) HandleDeleteClient(c echo.Context) error {
	ctx := c.Request().Context()
	clientIDParam := c.Param("client_id")

	reg, client, err := h.authenticateRegistrationRequest(c, clientIDParam)
	if err != nil {
		return err
	}

	// 登録メタデータ削除
	if err := h.regStore.Delete(ctx, reg.ID); err != nil {
		return registrationError(c, http.StatusInternalServerError, "server_error", "")
	}

	// クライアント論理削除
	if err := h.clientUpdater.SoftDelete(ctx, client.ID); err != nil {
		return registrationError(c, http.StatusInternalServerError, "server_error", "")
	}

	return c.NoContent(http.StatusNoContent)
}

// authenticateRegistrationRequest は Registration Access Token で認証し、
// 対応する ClientRegistration と Client を返す。
func (h *RegistrationHandler) authenticateRegistrationRequest(c echo.Context, clientID string) (*model.ClientRegistration, *model.Client, error) {
	ctx := c.Request().Context()

	bearerToken := extractBearerToken(c)
	if bearerToken == "" {
		return nil, nil, registrationError(c, http.StatusUnauthorized, "invalid_token", "registration access token required")
	}

	// client_id でクライアントを検索
	client, err := h.clientFinder.FindByClientIDWithRedirectURIs(ctx, clientID)
	if err != nil {
		return nil, nil, registrationError(c, http.StatusInternalServerError, "server_error", "")
	}
	if client == nil || client.Status != "active" {
		return nil, nil, registrationError(c, http.StatusNotFound, "invalid_client", "client not found")
	}

	// 登録情報を検索
	reg, err := h.regStore.FindByClientID(ctx, client.ID)
	if err != nil {
		return nil, nil, registrationError(c, http.StatusInternalServerError, "server_error", "")
	}
	if reg == nil {
		return nil, nil, registrationError(c, http.StatusNotFound, "invalid_client", "client not registered via dynamic registration")
	}

	// Registration Access Token の検証（タイミングセーフ比較）
	tokenHash := h.sha256Hex(bearerToken)
	if subtle.ConstantTimeCompare([]byte(tokenHash), []byte(reg.RegistrationAccessTokenHash)) != 1 {
		return nil, nil, registrationError(c, http.StatusUnauthorized, "invalid_token", "invalid registration access token")
	}

	return reg, client, nil
}

// validateRegistrationRequest は RFC 7591 のメタデータバリデーションを行う。
func (h *RegistrationHandler) validateRegistrationRequest(req *registrationRequest) *registrationErrorResponse {
	// redirect_uris は必須 (RFC 7591 Section 2)
	if len(req.RedirectURIs) == 0 {
		return &registrationErrorResponse{
			Error:            "invalid_redirect_uri",
			ErrorDescription: "redirect_uris is required",
		}
	}
	for _, uri := range req.RedirectURIs {
		if err := validateRegistrationRedirectURI(uri); err != nil {
			return &registrationErrorResponse{
				Error:            "invalid_redirect_uri",
				ErrorDescription: err.Error(),
			}
		}
	}

	// grant_types バリデーション
	for _, gt := range req.GrantTypes {
		if !supportedGrantTypes[gt] {
			return &registrationErrorResponse{
				Error:            "invalid_client_metadata",
				ErrorDescription: fmt.Sprintf("unsupported grant_type: %s", gt),
			}
		}
	}

	// response_types バリデーション
	for _, rt := range req.ResponseTypes {
		if !supportedResponseTypes[rt] {
			return &registrationErrorResponse{
				Error:            "invalid_client_metadata",
				ErrorDescription: fmt.Sprintf("unsupported response_type: %s", rt),
			}
		}
	}

	// token_endpoint_auth_method バリデーション
	if req.TokenEndpointAuthMethod != "" && !supportedAuthMethods[req.TokenEndpointAuthMethod] {
		return &registrationErrorResponse{
			Error:            "invalid_client_metadata",
			ErrorDescription: fmt.Sprintf("unsupported token_endpoint_auth_method: %s", req.TokenEndpointAuthMethod),
		}
	}

	// grant_types と response_types の整合性チェック
	grantTypes := req.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code"}
	}
	responseTypes := req.ResponseTypes
	if len(responseTypes) == 0 {
		responseTypes = []string{"code"}
	}
	hasAuthCodeGrant := false
	for _, gt := range grantTypes {
		if gt == "authorization_code" {
			hasAuthCodeGrant = true
			break
		}
	}
	hasCodeResponse := false
	for _, rt := range responseTypes {
		if rt == "code" {
			hasCodeResponse = true
			break
		}
	}
	if hasCodeResponse && !hasAuthCodeGrant {
		return &registrationErrorResponse{
			Error:            "invalid_client_metadata",
			ErrorDescription: "response_type 'code' requires grant_type 'authorization_code'",
		}
	}
	if hasAuthCodeGrant && !hasCodeResponse {
		return &registrationErrorResponse{
			Error:            "invalid_client_metadata",
			ErrorDescription: "grant_type 'authorization_code' requires response_type 'code'",
		}
	}

	return nil
}

// validateRegistrationRedirectURI は redirect_uri の形式を検証する (RFC 7591 + OIDC Core)。
func validateRegistrationRedirectURI(uri string) error {
	parsed, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("invalid URI: %s", uri)
	}
	if parsed.Fragment != "" {
		return fmt.Errorf("redirect URI must not contain a fragment: %s", uri)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("redirect URI must have scheme and host: %s", uri)
	}
	// HTTPS 必須（localhost は例外）
	if parsed.Scheme != "https" {
		host := parsed.Hostname()
		if host != "localhost" && host != "127.0.0.1" && host != "::1" {
			return fmt.Errorf("redirect URI must use HTTPS (except localhost): %s", uri)
		}
	}
	return nil
}

// extractBearerToken は Authorization ヘッダーから Bearer トークンを抽出する。
func extractBearerToken(c echo.Context) string {
	auth := c.Request().Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}

// registrationErrorResponse は RFC 7591 Section 3.2.2 のエラーレスポンス。
type registrationErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// registrationError は RFC 7591 形式のエラーレスポンスを返す。
func registrationError(c echo.Context, status int, errCode, description string) error {
	resp := registrationErrorResponse{Error: errCode}
	if description != "" {
		resp.ErrorDescription = description
	}
	return c.JSON(status, resp)
}

// generateRandomHex は暗号学的に安全なランダムバイトを hex エンコードして返す。
func generateRandomHex(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}
