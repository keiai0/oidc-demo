package management

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

var validGrantTypes = map[string]bool{
	"authorization_code": true,
	"refresh_token":      true,
	"client_credentials": true,
}

var validResponseTypes = map[string]bool{
	"code": true,
}

var validAuthMethods = map[string]bool{
	"client_secret_basic": true,
	"client_secret_post":  true,
	"none":                true,
}

// ClientHandler はクライアント管理の CRUD エンドポイントを処理する。
type ClientHandler struct {
	clientStore       ClientStore
	tenantStore       TenantStore
	tenantClientStore TenantClientStore
	hashPassword      HashPasswordFunc
}

// NewClientHandler は ClientHandler を生成する。
func NewClientHandler(
	clientStore ClientStore,
	tenantStore TenantStore,
	tenantClientStore TenantClientStore,
	hashPassword HashPasswordFunc,
) *ClientHandler {
	return &ClientHandler{
		clientStore:       clientStore,
		tenantStore:       tenantStore,
		tenantClientStore: tenantClientStore,
		hashPassword:      hashPassword,
	}
}

var validSubjectTypes = map[string]bool{
	"public":   true,
	"pairwise": true,
}

var validUserinfoSignedAlgs = map[string]bool{
	"RS256": true,
}

type createClientRequest struct {
	Name                      string   `json:"name"`
	GrantTypes                []string `json:"grant_types"`
	ResponseTypes             []string `json:"response_types"`
	TokenEndpointAuthMethod   string   `json:"token_endpoint_auth_method"`
	RequirePKCE               *bool    `json:"require_pkce,omitempty"`
	RedirectURIs              []string `json:"redirect_uris,omitempty"`
	PostLogoutRedirectURIs    []string `json:"post_logout_redirect_uris,omitempty"`
	FrontchannelLogoutURI     *string  `json:"frontchannel_logout_uri,omitempty"`
	BackchannelLogoutURI      *string  `json:"backchannel_logout_uri,omitempty"`
	SubjectType               string   `json:"subject_type,omitempty"`
	SectorIdentifierURI       *string  `json:"sector_identifier_uri,omitempty"`
	UserinfoSignedResponseAlg *string  `json:"userinfo_signed_response_alg,omitempty"`
}

type updateClientRequest struct {
	Name                      *string  `json:"name,omitempty"`
	GrantTypes                []string `json:"grant_types,omitempty"`
	ResponseTypes             []string `json:"response_types,omitempty"`
	TokenEndpointAuthMethod   *string  `json:"token_endpoint_auth_method,omitempty"`
	RequirePKCE               *bool    `json:"require_pkce,omitempty"`
	FrontchannelLogoutURI     *string  `json:"frontchannel_logout_uri,omitempty"`
	BackchannelLogoutURI      *string  `json:"backchannel_logout_uri,omitempty"`
	SubjectType               *string  `json:"subject_type,omitempty"`
	SectorIdentifierURI       *string  `json:"sector_identifier_uri,omitempty"`
	UserinfoSignedResponseAlg *string  `json:"userinfo_signed_response_alg,omitempty"`
}

type clientResponse struct {
	ID                        string   `json:"id"`
	ClientID                  string   `json:"client_id"`
	Name                      string   `json:"name"`
	GrantTypes                []string `json:"grant_types"`
	ResponseTypes             []string `json:"response_types"`
	TokenEndpointAuthMethod   string   `json:"token_endpoint_auth_method"`
	RequirePKCE               bool     `json:"require_pkce"`
	FrontchannelLogoutURI     *string  `json:"frontchannel_logout_uri,omitempty"`
	BackchannelLogoutURI      *string  `json:"backchannel_logout_uri,omitempty"`
	SubjectType               string   `json:"subject_type"`
	SectorIdentifierURI       *string  `json:"sector_identifier_uri,omitempty"`
	UserinfoSignedResponseAlg *string  `json:"userinfo_signed_response_alg,omitempty"`
	Status                    string   `json:"status"`
	CreatedAt                 string   `json:"created_at"`
	UpdatedAt                 string   `json:"updated_at"`
}

type clientCreateResponse struct {
	clientResponse
	ClientSecret string `json:"client_secret"`
}

func toClientResponse(c *model.Client) clientResponse {
	return clientResponse{
		ID:                        c.ID.String(),
		ClientID:                  c.ClientID,
		Name:                      c.Name,
		GrantTypes:                []string(c.GrantTypes),
		ResponseTypes:             []string(c.ResponseTypes),
		TokenEndpointAuthMethod:   c.TokenEndpointAuthMethod,
		RequirePKCE:               c.RequirePKCE,
		FrontchannelLogoutURI:     c.FrontchannelLogoutURI,
		BackchannelLogoutURI:      c.BackchannelLogoutURI,
		SubjectType:               c.SubjectType,
		SectorIdentifierURI:       c.SectorIdentifierURI,
		UserinfoSignedResponseAlg: c.UserinfoSignedResponseAlg,
		Status:                    c.Status,
		CreatedAt:                 c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:                 c.UpdatedAt.Format(time.RFC3339),
	}
}

// HandleList は GET /management/v1/tenants/:tenant_id/clients を処理する。
func (h *ClientHandler) HandleList(c echo.Context) error {
	ctx := c.Request().Context()

	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		return badRequest(c, "invalid tenant_id format")
	}

	tenant, err := h.tenantStore.FindByID(ctx, tenantID)
	if err != nil {
		c.Logger().Errorf("failed to find tenant: %v", err)
		return serverError(c)
	}
	if tenant == nil {
		return notFound(c, "tenant not found")
	}

	p := parsePagination(c)
	clients, total, err := h.clientStore.ListByTenantID(ctx, tenantID, p.Limit, p.Offset)
	if err != nil {
		c.Logger().Errorf("failed to list clients: %v", err)
		return serverError(c)
	}

	data := make([]clientResponse, len(clients))
	for i, cl := range clients {
		data[i] = toClientResponse(&cl)
	}

	return c.JSON(http.StatusOK, ListResponse[clientResponse]{
		Data:       data,
		TotalCount: total,
	})
}

// HandleCreate は POST /management/v1/tenants/:tenant_id/clients を処理する。
func (h *ClientHandler) HandleCreate(c echo.Context) error {
	ctx := c.Request().Context()

	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		return badRequest(c, "invalid tenant_id format")
	}

	tenant, err := h.tenantStore.FindByID(ctx, tenantID)
	if err != nil {
		c.Logger().Errorf("failed to find tenant: %v", err)
		return serverError(c)
	}
	if tenant == nil {
		return notFound(c, "tenant not found")
	}

	var req createClientRequest
	if err := c.Bind(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	if req.Name == "" || len(req.Name) > 255 {
		return badRequest(c, "name is required and must be at most 255 characters")
	}
	if len(req.GrantTypes) == 0 {
		return badRequest(c, "grant_types is required")
	}
	for _, gt := range req.GrantTypes {
		if !validGrantTypes[gt] {
			return badRequest(c, "unsupported grant_type: "+gt)
		}
	}
	if len(req.ResponseTypes) == 0 {
		return badRequest(c, "response_types is required")
	}
	for _, rt := range req.ResponseTypes {
		if !validResponseTypes[rt] {
			return badRequest(c, "unsupported response_type: "+rt)
		}
	}
	if req.TokenEndpointAuthMethod == "" {
		req.TokenEndpointAuthMethod = "client_secret_basic"
	}
	if !validAuthMethods[req.TokenEndpointAuthMethod] {
		return badRequest(c, "unsupported token_endpoint_auth_method: "+req.TokenEndpointAuthMethod)
	}
	for _, uri := range req.RedirectURIs {
		if err := validateRedirectURI(uri); err != nil {
			return badRequest(c, err.Error())
		}
	}
	for _, uri := range req.PostLogoutRedirectURIs {
		if err := validateRedirectURI(uri); err != nil {
			return badRequest(c, err.Error())
		}
	}

	clientID, err := generateClientID()
	if err != nil {
		c.Logger().Errorf("failed to generate client_id: %v", err)
		return serverError(c)
	}
	clientSecret, err := generateClientSecret()
	if err != nil {
		c.Logger().Errorf("failed to generate client_secret: %v", err)
		return serverError(c)
	}
	secretHash, err := h.hashPassword(clientSecret)
	if err != nil {
		c.Logger().Errorf("failed to hash client_secret: %v", err)
		return serverError(c)
	}

	requirePKCE := true
	if req.RequirePKCE != nil {
		requirePKCE = *req.RequirePKCE
	}

	// subject_type バリデーション (OIDC Core Section 8)
	subjectType := "public"
	if req.SubjectType != "" {
		if !validSubjectTypes[req.SubjectType] {
			return badRequest(c, "subject_type must be 'public' or 'pairwise'")
		}
		subjectType = req.SubjectType
	}

	// userinfo_signed_response_alg バリデーション (OIDC Core Section 5.3.2)
	if req.UserinfoSignedResponseAlg != nil && *req.UserinfoSignedResponseAlg != "" {
		if !validUserinfoSignedAlgs[*req.UserinfoSignedResponseAlg] {
			return badRequest(c, "unsupported userinfo_signed_response_alg (supported: RS256)")
		}
	}

	// Pairwise: ソルト自動生成
	var pairwiseSalt *string
	if subjectType == "pairwise" {
		salt, err := generatePairwiseSalt()
		if err != nil {
			c.Logger().Errorf("failed to generate pairwise salt: %v", err)
			return serverError(c)
		}
		pairwiseSalt = &salt
	}

	client := &model.Client{
		ClientID:                  clientID,
		ClientSecretHash:          secretHash,
		Name:                      req.Name,
		GrantTypes:                model.StringSlice(req.GrantTypes),
		ResponseTypes:             model.StringSlice(req.ResponseTypes),
		TokenEndpointAuthMethod:   req.TokenEndpointAuthMethod,
		RequirePKCE:               requirePKCE,
		FrontchannelLogoutURI:     req.FrontchannelLogoutURI,
		BackchannelLogoutURI:      req.BackchannelLogoutURI,
		SubjectType:               subjectType,
		SectorIdentifierURI:       req.SectorIdentifierURI,
		PairwiseSalt:              pairwiseSalt,
		UserinfoSignedResponseAlg: req.UserinfoSignedResponseAlg,
		Status:                    "active",
	}

	// Redirect URI を関連として設定
	for _, uri := range req.RedirectURIs {
		client.RedirectURIs = append(client.RedirectURIs, model.RedirectURI{URI: uri})
	}
	for _, uri := range req.PostLogoutRedirectURIs {
		client.PostLogoutRedirectURIs = append(client.PostLogoutRedirectURIs, model.PostLogoutRedirectURI{URI: uri})
	}

	if err := h.clientStore.Create(ctx, client); err != nil {
		c.Logger().Errorf("failed to create client: %v", err)
		return serverError(c)
	}

	// 中間テーブルにテナント-クライアント紐づけを追加
	tc := &model.TenantClient{
		TenantID: tenantID,
		ClientID: client.ID,
		Enabled:  true,
	}
	if err := h.tenantClientStore.Create(ctx, tc); err != nil {
		c.Logger().Errorf("failed to create tenant-client association: %v", err)
		return serverError(c)
	}

	resp := clientCreateResponse{
		clientResponse: toClientResponse(client),
		ClientSecret:   clientSecret,
	}

	return c.JSON(http.StatusCreated, resp)
}

// HandleGet は GET /management/v1/clients/:id を処理する。
func (h *ClientHandler) HandleGet(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return badRequest(c, "invalid client id format")
	}

	client, err := h.clientStore.FindByIDWithRelations(ctx, id)
	if err != nil {
		c.Logger().Errorf("failed to find client: %v", err)
		return serverError(c)
	}
	if client == nil {
		return notFound(c, "client not found")
	}

	type clientDetailResponse struct {
		clientResponse
		RedirectURIs           []redirectURIResponse `json:"redirect_uris"`
		PostLogoutRedirectURIs []redirectURIResponse `json:"post_logout_redirect_uris"`
	}

	redirectURIs := make([]redirectURIResponse, len(client.RedirectURIs))
	for i, ru := range client.RedirectURIs {
		redirectURIs[i] = redirectURIResponse{
			ID:        ru.ID.String(),
			URI:       ru.URI,
			CreatedAt: ru.CreatedAt.Format(time.RFC3339),
		}
	}

	postLogoutURIs := make([]redirectURIResponse, len(client.PostLogoutRedirectURIs))
	for i, ru := range client.PostLogoutRedirectURIs {
		postLogoutURIs[i] = redirectURIResponse{
			ID:        ru.ID.String(),
			URI:       ru.URI,
			CreatedAt: ru.CreatedAt.Format(time.RFC3339),
		}
	}

	return c.JSON(http.StatusOK, clientDetailResponse{
		clientResponse:         toClientResponse(client),
		RedirectURIs:           redirectURIs,
		PostLogoutRedirectURIs: postLogoutURIs,
	})
}

// HandleUpdate は PUT /management/v1/clients/:id を処理する。
func (h *ClientHandler) HandleUpdate(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return badRequest(c, "invalid client id format")
	}

	client, err := h.clientStore.FindByID(ctx, id)
	if err != nil {
		c.Logger().Errorf("failed to find client: %v", err)
		return serverError(c)
	}
	if client == nil {
		return notFound(c, "client not found")
	}

	var req updateClientRequest
	if err := c.Bind(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	if req.Name != nil {
		if *req.Name == "" || len(*req.Name) > 255 {
			return badRequest(c, "name must be 1-255 characters")
		}
		client.Name = *req.Name
	}
	if req.GrantTypes != nil {
		for _, gt := range req.GrantTypes {
			if !validGrantTypes[gt] {
				return badRequest(c, "unsupported grant_type: "+gt)
			}
		}
		client.GrantTypes = model.StringSlice(req.GrantTypes)
	}
	if req.ResponseTypes != nil {
		for _, rt := range req.ResponseTypes {
			if !validResponseTypes[rt] {
				return badRequest(c, "unsupported response_type: "+rt)
			}
		}
		client.ResponseTypes = model.StringSlice(req.ResponseTypes)
	}
	if req.TokenEndpointAuthMethod != nil {
		if !validAuthMethods[*req.TokenEndpointAuthMethod] {
			return badRequest(c, "unsupported token_endpoint_auth_method: "+*req.TokenEndpointAuthMethod)
		}
		client.TokenEndpointAuthMethod = *req.TokenEndpointAuthMethod
	}
	if req.RequirePKCE != nil {
		client.RequirePKCE = *req.RequirePKCE
	}
	if req.FrontchannelLogoutURI != nil {
		client.FrontchannelLogoutURI = req.FrontchannelLogoutURI
	}
	if req.BackchannelLogoutURI != nil {
		client.BackchannelLogoutURI = req.BackchannelLogoutURI
	}
	if req.SubjectType != nil {
		if !validSubjectTypes[*req.SubjectType] {
			return badRequest(c, "subject_type must be 'public' or 'pairwise'")
		}
		client.SubjectType = *req.SubjectType
		// pairwise に変更された場合、ソルトが未設定なら自動生成
		if *req.SubjectType == "pairwise" && client.PairwiseSalt == nil {
			salt, err := generatePairwiseSalt()
			if err != nil {
				c.Logger().Errorf("failed to generate pairwise salt: %v", err)
				return serverError(c)
			}
			client.PairwiseSalt = &salt
		}
	}
	if req.SectorIdentifierURI != nil {
		client.SectorIdentifierURI = req.SectorIdentifierURI
	}
	if req.UserinfoSignedResponseAlg != nil {
		if *req.UserinfoSignedResponseAlg != "" && !validUserinfoSignedAlgs[*req.UserinfoSignedResponseAlg] {
			return badRequest(c, "unsupported userinfo_signed_response_alg (supported: RS256)")
		}
		client.UserinfoSignedResponseAlg = req.UserinfoSignedResponseAlg
	}

	if err := h.clientStore.Update(ctx, client); err != nil {
		c.Logger().Errorf("failed to update client: %v", err)
		return serverError(c)
	}

	return c.JSON(http.StatusOK, toClientResponse(client))
}

// HandleDelete は DELETE /management/v1/clients/:id を処理する。
func (h *ClientHandler) HandleDelete(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return badRequest(c, "invalid client id format")
	}

	client, err := h.clientStore.FindByID(ctx, id)
	if err != nil {
		c.Logger().Errorf("failed to find client: %v", err)
		return serverError(c)
	}
	if client == nil {
		return notFound(c, "client not found")
	}

	if err := h.clientStore.SoftDelete(ctx, id); err != nil {
		c.Logger().Errorf("failed to delete client: %v", err)
		return serverError(c)
	}

	return c.NoContent(http.StatusNoContent)
}

// HandleRotateSecret は PUT /management/v1/clients/:id/secret を処理する。
func (h *ClientHandler) HandleRotateSecret(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return badRequest(c, "invalid client id format")
	}

	client, err := h.clientStore.FindByID(ctx, id)
	if err != nil {
		c.Logger().Errorf("failed to find client: %v", err)
		return serverError(c)
	}
	if client == nil {
		return notFound(c, "client not found")
	}

	newSecret, err := generateClientSecret()
	if err != nil {
		c.Logger().Errorf("failed to generate new secret: %v", err)
		return serverError(c)
	}
	newHash, err := h.hashPassword(newSecret)
	if err != nil {
		c.Logger().Errorf("failed to hash new secret: %v", err)
		return serverError(c)
	}

	if err := h.clientStore.UpdateSecretHash(ctx, id, newHash); err != nil {
		c.Logger().Errorf("failed to update secret: %v", err)
		return serverError(c)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"client_id":     client.ClientID,
		"client_secret": newSecret,
	})
}

// HandleListAll は GET /management/v1/clients を処理する（全クライアント一覧）。
func (h *ClientHandler) HandleListAll(c echo.Context) error {
	ctx := c.Request().Context()

	p := parsePagination(c)
	clients, total, err := h.clientStore.List(ctx, p.Limit, p.Offset)
	if err != nil {
		c.Logger().Errorf("failed to list all clients: %v", err)
		return serverError(c)
	}

	data := make([]clientResponse, len(clients))
	for i, cl := range clients {
		data[i] = toClientResponse(&cl)
	}

	return c.JSON(http.StatusOK, ListResponse[clientResponse]{
		Data:       data,
		TotalCount: total,
	})
}

type addTenantRequest struct {
	TenantID string `json:"tenant_id"`
}

type tenantAssociationResponse struct {
	TenantID   string `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
	TenantCode string `json:"tenant_code"`
	Enabled    bool   `json:"enabled"`
	CreatedAt  string `json:"created_at"`
}

// HandleAddTenant は POST /management/v1/clients/:id/tenants を処理する。
func (h *ClientHandler) HandleAddTenant(c echo.Context) error {
	ctx := c.Request().Context()

	clientDBID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return badRequest(c, "invalid client id format")
	}

	client, err := h.clientStore.FindByID(ctx, clientDBID)
	if err != nil {
		c.Logger().Errorf("failed to find client: %v", err)
		return serverError(c)
	}
	if client == nil {
		return notFound(c, "client not found")
	}

	var req addTenantRequest
	if err := c.Bind(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return badRequest(c, "invalid tenant_id format")
	}

	tenant, err := h.tenantStore.FindByID(ctx, tenantID)
	if err != nil {
		c.Logger().Errorf("failed to find tenant: %v", err)
		return serverError(c)
	}
	if tenant == nil {
		return notFound(c, "tenant not found")
	}

	// 既に紐づいているか確認
	exists, err := h.tenantClientStore.ExistsByTenantAndClient(ctx, tenantID, clientDBID)
	if err != nil {
		c.Logger().Errorf("failed to check tenant-client association: %v", err)
		return serverError(c)
	}
	if exists {
		return c.JSON(http.StatusConflict, map[string]string{"error": "association already exists"})
	}

	tc := &model.TenantClient{
		TenantID: tenantID,
		ClientID: clientDBID,
		Enabled:  true,
	}
	if err := h.tenantClientStore.Create(ctx, tc); err != nil {
		c.Logger().Errorf("failed to create tenant-client association: %v", err)
		return serverError(c)
	}

	return c.JSON(http.StatusCreated, tenantAssociationResponse{
		TenantID:   tenant.ID.String(),
		TenantName: tenant.Name,
		TenantCode: tenant.Code,
		Enabled:    tc.Enabled,
		CreatedAt:  tc.CreatedAt.Format(time.RFC3339),
	})
}

// HandleRemoveTenant は DELETE /management/v1/clients/:id/tenants/:tenant_id を処理する。
func (h *ClientHandler) HandleRemoveTenant(c echo.Context) error {
	ctx := c.Request().Context()

	clientDBID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return badRequest(c, "invalid client id format")
	}

	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		return badRequest(c, "invalid tenant_id format")
	}

	if err := h.tenantClientStore.Delete(ctx, tenantID, clientDBID); err != nil {
		c.Logger().Errorf("failed to delete tenant-client association: %v", err)
		return notFound(c, "association not found")
	}

	return c.NoContent(http.StatusNoContent)
}

// HandleListTenants は GET /management/v1/clients/:id/tenants を処理する。
func (h *ClientHandler) HandleListTenants(c echo.Context) error {
	ctx := c.Request().Context()

	clientDBID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return badRequest(c, "invalid client id format")
	}

	client, err := h.clientStore.FindByID(ctx, clientDBID)
	if err != nil {
		c.Logger().Errorf("failed to find client: %v", err)
		return serverError(c)
	}
	if client == nil {
		return notFound(c, "client not found")
	}

	tcs, err := h.tenantClientStore.ListByClientID(ctx, clientDBID)
	if err != nil {
		c.Logger().Errorf("failed to list tenant-client associations: %v", err)
		return serverError(c)
	}

	data := make([]tenantAssociationResponse, len(tcs))
	for i, tc := range tcs {
		data[i] = tenantAssociationResponse{
			TenantID:   tc.TenantID.String(),
			TenantName: tc.Tenant.Name,
			TenantCode: tc.Tenant.Code,
			Enabled:    tc.Enabled,
			CreatedAt:  tc.CreatedAt.Format(time.RFC3339),
		}
	}

	return c.JSON(http.StatusOK, data)
}

// validateRedirectURI は URI が有効でフラグメントを含まないことを検証する（RFC 6749 Section 3.1.2）。
func validateRedirectURI(uri string) error {
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
	return nil
}
