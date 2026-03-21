package management

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// FederationProviderStore は federation_providers の管理操作を定義する。
type FederationProviderStore interface {
	Create(ctx context.Context, provider *model.FederationProvider) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.FederationProvider, error)
	ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]model.FederationProvider, error)
	Update(ctx context.Context, provider *model.FederationProvider) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type EncryptFunc func(plaintext []byte, key []byte) (string, error)

// FederationProviderHandler は管理 API の federation provider CRUD を処理する。
type FederationProviderHandler struct {
	providerStore FederationProviderStore
	encrypt       EncryptFunc
	encKey        []byte
}

// NewFederationProviderHandler は FederationProviderHandler を生成する。
func NewFederationProviderHandler(
	providerStore FederationProviderStore,
	encrypt EncryptFunc,
	encKey []byte,
) *FederationProviderHandler {
	return &FederationProviderHandler{
		providerStore: providerStore,
		encrypt:       encrypt,
		encKey:        encKey,
	}
}

type federationProviderResponse struct {
	ID            string `json:"id"`
	TenantID      string `json:"tenant_id"`
	Name          string `json:"name"`
	Issuer        string `json:"issuer"`
	ClientID      string `json:"client_id"`
	Scopes        string `json:"scopes"`
	AutoProvision bool   `json:"auto_provision"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

func toFederationProviderResponse(p *model.FederationProvider) federationProviderResponse {
	return federationProviderResponse{
		ID:            p.ID.String(),
		TenantID:      p.TenantID.String(),
		Name:          p.Name,
		Issuer:        p.Issuer,
		ClientID:      p.ClientID,
		Scopes:        p.Scopes,
		AutoProvision: p.AutoProvision,
		Status:        p.Status,
		CreatedAt:     p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// HandleList は GET /management/v1/tenants/:tenant_id/federation-providers を処理する。
func (h *FederationProviderHandler) HandleList(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}

	providers, err := h.providerStore.ListByTenantID(c.Request().Context(), tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	items := make([]federationProviderResponse, len(providers))
	for i, p := range providers {
		items[i] = toFederationProviderResponse(&p)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"data": items})
}

type createFederationProviderRequest struct {
	Name          string `json:"name"`
	Issuer        string `json:"issuer"`
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	Scopes        string `json:"scopes"`
	AutoProvision *bool  `json:"auto_provision"`
}

// HandleCreate は POST /management/v1/tenants/:tenant_id/federation-providers を処理する。
func (h *FederationProviderHandler) HandleCreate(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id"})
	}

	var req createFederationProviderRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
	}

	if req.Name == "" || req.Issuer == "" || req.ClientID == "" || req.ClientSecret == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name, issuer, client_id, client_secret are required"})
	}

	// client_secret を AES-256-GCM で暗号化
	encrypted, err := h.encrypt([]byte(req.ClientSecret), h.encKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to encrypt client_secret"})
	}

	scopes := req.Scopes
	if scopes == "" {
		scopes = "openid profile email"
	}

	autoProvision := true
	if req.AutoProvision != nil {
		autoProvision = *req.AutoProvision
	}

	provider := &model.FederationProvider{
		TenantID:        tenantID,
		Name:            req.Name,
		Issuer:          req.Issuer,
		ClientID:        req.ClientID,
		ClientSecretEnc: encrypted,
		Scopes:          scopes,
		AutoProvision:   autoProvision,
		Status:          "active",
	}

	if err := h.providerStore.Create(c.Request().Context(), provider); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	return c.JSON(http.StatusCreated, toFederationProviderResponse(provider))
}

// HandleGet は GET /management/v1/federation-providers/:id を処理する。
func (h *FederationProviderHandler) HandleGet(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	provider, err := h.providerStore.FindByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	if provider == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not_found"})
	}

	return c.JSON(http.StatusOK, toFederationProviderResponse(provider))
}

type updateFederationProviderRequest struct {
	Issuer        *string `json:"issuer"`
	ClientID      *string `json:"client_id"`
	ClientSecret  *string `json:"client_secret"`
	Scopes        *string `json:"scopes"`
	AutoProvision *bool   `json:"auto_provision"`
	Status        *string `json:"status"`
}

// HandleUpdate は PUT /management/v1/federation-providers/:id を処理する。
func (h *FederationProviderHandler) HandleUpdate(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	ctx := c.Request().Context()
	provider, err := h.providerStore.FindByID(ctx, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	if provider == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not_found"})
	}

	var req updateFederationProviderRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
	}

	if req.Issuer != nil {
		provider.Issuer = *req.Issuer
	}
	if req.ClientID != nil {
		provider.ClientID = *req.ClientID
	}
	if req.ClientSecret != nil {
		encrypted, err := h.encrypt([]byte(*req.ClientSecret), h.encKey)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to encrypt client_secret"})
		}
		provider.ClientSecretEnc = encrypted
	}
	if req.Scopes != nil {
		provider.Scopes = *req.Scopes
	}
	if req.AutoProvision != nil {
		provider.AutoProvision = *req.AutoProvision
	}
	if req.Status != nil {
		provider.Status = *req.Status
	}

	if err := h.providerStore.Update(ctx, provider); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	return c.JSON(http.StatusOK, toFederationProviderResponse(provider))
}

// HandleDelete は DELETE /management/v1/federation-providers/:id を処理する。
func (h *FederationProviderHandler) HandleDelete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid id"})
	}

	if err := h.providerStore.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	return c.NoContent(http.StatusNoContent)
}
