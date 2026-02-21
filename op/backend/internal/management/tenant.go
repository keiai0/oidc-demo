package management

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

var tenantCodeRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)

// TenantHandler はテナント管理の CRUD エンドポイントを処理する。
type TenantHandler struct {
	tenantStore TenantStore
}

// NewTenantHandler は TenantHandler を生成する。
func NewTenantHandler(store TenantStore) *TenantHandler {
	return &TenantHandler{tenantStore: store}
}

type createTenantRequest struct {
	Code                 string `json:"code"`
	Name                 string `json:"name"`
	SessionLifetime      *int   `json:"session_lifetime,omitempty"`
	AuthCodeLifetime     *int   `json:"auth_code_lifetime,omitempty"`
	AccessTokenLifetime  *int   `json:"access_token_lifetime,omitempty"`
	RefreshTokenLifetime *int   `json:"refresh_token_lifetime,omitempty"`
	IDTokenLifetime      *int   `json:"id_token_lifetime,omitempty"`
}

type updateTenantRequest struct {
	Name                 *string `json:"name,omitempty"`
	SessionLifetime      *int    `json:"session_lifetime,omitempty"`
	AuthCodeLifetime     *int    `json:"auth_code_lifetime,omitempty"`
	AccessTokenLifetime  *int    `json:"access_token_lifetime,omitempty"`
	RefreshTokenLifetime *int    `json:"refresh_token_lifetime,omitempty"`
	IDTokenLifetime      *int    `json:"id_token_lifetime,omitempty"`
}

type tenantResponse struct {
	ID                   string `json:"id"`
	Code                 string `json:"code"`
	Name                 string `json:"name"`
	SessionLifetime      int    `json:"session_lifetime"`
	AuthCodeLifetime     int    `json:"auth_code_lifetime"`
	AccessTokenLifetime  int    `json:"access_token_lifetime"`
	RefreshTokenLifetime int    `json:"refresh_token_lifetime"`
	IDTokenLifetime      int    `json:"id_token_lifetime"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
}

func toTenantResponse(t *model.Tenant) tenantResponse {
	return tenantResponse{
		ID:                   t.ID.String(),
		Code:                 t.Code,
		Name:                 t.Name,
		SessionLifetime:      t.SessionLifetime,
		AuthCodeLifetime:     t.AuthCodeLifetime,
		AccessTokenLifetime:  t.AccessTokenLifetime,
		RefreshTokenLifetime: t.RefreshTokenLifetime,
		IDTokenLifetime:      t.IDTokenLifetime,
		CreatedAt:            t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            t.UpdatedAt.Format(time.RFC3339),
	}
}

// HandleList は GET /management/v1/tenants を処理する。
func (h *TenantHandler) HandleList(c echo.Context) error {
	ctx := c.Request().Context()
	p := parsePagination(c)

	tenants, total, err := h.tenantStore.List(ctx, p.Limit, p.Offset)
	if err != nil {
		c.Logger().Errorf("failed to list tenants: %v", err)
		return serverError(c)
	}

	data := make([]tenantResponse, len(tenants))
	for i, t := range tenants {
		data[i] = toTenantResponse(&t)
	}

	return c.JSON(http.StatusOK, ListResponse[tenantResponse]{
		Data:       data,
		TotalCount: total,
	})
}

// HandleCreate は POST /management/v1/tenants を処理する。
func (h *TenantHandler) HandleCreate(c echo.Context) error {
	ctx := c.Request().Context()

	var req createTenantRequest
	if err := c.Bind(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	if req.Code == "" {
		return badRequest(c, "code is required")
	}
	if !tenantCodeRegex.MatchString(req.Code) {
		return badRequest(c, "code must be 3-63 chars, lowercase alphanumeric and hyphens, start/end with alphanumeric")
	}
	if req.Name == "" {
		return badRequest(c, "name is required")
	}
	if len(req.Name) > 255 {
		return badRequest(c, "name must be at most 255 characters")
	}
	if err := validateLifetimes(req.SessionLifetime, req.AuthCodeLifetime, req.AccessTokenLifetime, req.RefreshTokenLifetime, req.IDTokenLifetime); err != nil {
		return badRequest(c, err.Error())
	}

	// 重複チェック
	existing, err := h.tenantStore.FindByCode(ctx, req.Code)
	if err != nil {
		c.Logger().Errorf("failed to check tenant code: %v", err)
		return serverError(c)
	}
	if existing != nil {
		return conflict(c, "tenant code already exists")
	}

	tenant := &model.Tenant{
		Code:                 req.Code,
		Name:                 req.Name,
		SessionLifetime:      orDefault(req.SessionLifetime, 3600),
		AuthCodeLifetime:     orDefault(req.AuthCodeLifetime, 60),
		AccessTokenLifetime:  orDefault(req.AccessTokenLifetime, 3600),
		RefreshTokenLifetime: orDefault(req.RefreshTokenLifetime, 2592000),
		IDTokenLifetime:      orDefault(req.IDTokenLifetime, 3600),
	}

	if err := h.tenantStore.Create(ctx, tenant); err != nil {
		c.Logger().Errorf("failed to create tenant: %v", err)
		return serverError(c)
	}

	return c.JSON(http.StatusCreated, toTenantResponse(tenant))
}

// HandleGet は GET /management/v1/tenants/:tenant_id を処理する。
func (h *TenantHandler) HandleGet(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		return badRequest(c, "invalid tenant_id format")
	}

	tenant, err := h.tenantStore.FindByID(ctx, id)
	if err != nil {
		c.Logger().Errorf("failed to find tenant: %v", err)
		return serverError(c)
	}
	if tenant == nil {
		return notFound(c, "tenant not found")
	}

	return c.JSON(http.StatusOK, toTenantResponse(tenant))
}

// HandleUpdate は PUT /management/v1/tenants/:tenant_id を処理する。
func (h *TenantHandler) HandleUpdate(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		return badRequest(c, "invalid tenant_id format")
	}

	tenant, err := h.tenantStore.FindByID(ctx, id)
	if err != nil {
		c.Logger().Errorf("failed to find tenant: %v", err)
		return serverError(c)
	}
	if tenant == nil {
		return notFound(c, "tenant not found")
	}

	var req updateTenantRequest
	if err := c.Bind(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	if err := validateLifetimes(req.SessionLifetime, req.AuthCodeLifetime, req.AccessTokenLifetime, req.RefreshTokenLifetime, req.IDTokenLifetime); err != nil {
		return badRequest(c, err.Error())
	}

	// 非nil フィールドのみ適用 (partial update)
	if req.Name != nil {
		if *req.Name == "" || len(*req.Name) > 255 {
			return badRequest(c, "name must be 1-255 characters")
		}
		tenant.Name = *req.Name
	}
	if req.SessionLifetime != nil {
		tenant.SessionLifetime = *req.SessionLifetime
	}
	if req.AuthCodeLifetime != nil {
		tenant.AuthCodeLifetime = *req.AuthCodeLifetime
	}
	if req.AccessTokenLifetime != nil {
		tenant.AccessTokenLifetime = *req.AccessTokenLifetime
	}
	if req.RefreshTokenLifetime != nil {
		tenant.RefreshTokenLifetime = *req.RefreshTokenLifetime
	}
	if req.IDTokenLifetime != nil {
		tenant.IDTokenLifetime = *req.IDTokenLifetime
	}

	if err := h.tenantStore.Update(ctx, tenant); err != nil {
		c.Logger().Errorf("failed to update tenant: %v", err)
		return serverError(c)
	}

	return c.JSON(http.StatusOK, toTenantResponse(tenant))
}

func orDefault(v *int, def int) int {
	if v != nil {
		return *v
	}
	return def
}

func validateLifetimes(vals ...*int) error {
	for _, v := range vals {
		if v != nil && *v <= 0 {
			return fmt.Errorf("lifetime values must be positive")
		}
	}
	return nil
}
