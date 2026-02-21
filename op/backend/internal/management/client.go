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
	clientStore  ClientStore
	tenantStore  TenantStore
	hashPassword HashPasswordFunc
}

// NewClientHandler は ClientHandler を生成する。
func NewClientHandler(
	clientStore ClientStore,
	tenantStore TenantStore,
	hashPassword HashPasswordFunc,
) *ClientHandler {
	return &ClientHandler{
		clientStore:  clientStore,
		tenantStore:  tenantStore,
		hashPassword: hashPassword,
	}
}

type createClientRequest struct {
	Name                    string   `json:"name"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	RequirePKCE             *bool    `json:"require_pkce,omitempty"`
	RedirectURIs            []string `json:"redirect_uris,omitempty"`
	PostLogoutRedirectURIs  []string `json:"post_logout_redirect_uris,omitempty"`
	FrontchannelLogoutURI   *string  `json:"frontchannel_logout_uri,omitempty"`
	BackchannelLogoutURI    *string  `json:"backchannel_logout_uri,omitempty"`
}

type updateClientRequest struct {
	Name                    *string  `json:"name,omitempty"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	TokenEndpointAuthMethod *string  `json:"token_endpoint_auth_method,omitempty"`
	RequirePKCE             *bool    `json:"require_pkce,omitempty"`
	FrontchannelLogoutURI   *string  `json:"frontchannel_logout_uri,omitempty"`
	BackchannelLogoutURI    *string  `json:"backchannel_logout_uri,omitempty"`
}

type clientResponse struct {
	ID                      string   `json:"id"`
	TenantID                string   `json:"tenant_id"`
	ClientID                string   `json:"client_id"`
	Name                    string   `json:"name"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	RequirePKCE             bool     `json:"require_pkce"`
	FrontchannelLogoutURI   *string  `json:"frontchannel_logout_uri,omitempty"`
	BackchannelLogoutURI    *string  `json:"backchannel_logout_uri,omitempty"`
	Status                  string   `json:"status"`
	CreatedAt               string   `json:"created_at"`
	UpdatedAt               string   `json:"updated_at"`
}

type clientCreateResponse struct {
	clientResponse
	ClientSecret string `json:"client_secret"`
}

func toClientResponse(c *model.Client) clientResponse {
	return clientResponse{
		ID:                      c.ID.String(),
		TenantID:                c.TenantID.String(),
		ClientID:                c.ClientID,
		Name:                    c.Name,
		GrantTypes:              []string(c.GrantTypes),
		ResponseTypes:           []string(c.ResponseTypes),
		TokenEndpointAuthMethod: c.TokenEndpointAuthMethod,
		RequirePKCE:             c.RequirePKCE,
		FrontchannelLogoutURI:   c.FrontchannelLogoutURI,
		BackchannelLogoutURI:    c.BackchannelLogoutURI,
		Status:                  c.Status,
		CreatedAt:               c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:               c.UpdatedAt.Format(time.RFC3339),
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

	client := &model.Client{
		TenantID:                tenantID,
		ClientID:                clientID,
		ClientSecretHash:        secretHash,
		Name:                    req.Name,
		GrantTypes:              model.StringSlice(req.GrantTypes),
		ResponseTypes:           model.StringSlice(req.ResponseTypes),
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		RequirePKCE:             requirePKCE,
		FrontchannelLogoutURI:   req.FrontchannelLogoutURI,
		BackchannelLogoutURI:    req.BackchannelLogoutURI,
		Status:                  "active",
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
