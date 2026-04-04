package management

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// AuthorizationDetailTypeStore は認可詳細タイプの永続化操作を定義する (RFC 9396)。
type AuthorizationDetailTypeStore interface {
	Create(ctx context.Context, adt *model.AuthorizationDetailType) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.AuthorizationDetailType, error)
	ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]model.AuthorizationDetailType, error)
	Update(ctx context.Context, adt *model.AuthorizationDetailType) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// AuthorizationDetailTypeHandler は認可詳細タイプの管理 API ハンドラ。
type AuthorizationDetailTypeHandler struct {
	store AuthorizationDetailTypeStore
}

// NewAuthorizationDetailTypeHandler は AuthorizationDetailTypeHandler を生成する。
func NewAuthorizationDetailTypeHandler(store AuthorizationDetailTypeStore) *AuthorizationDetailTypeHandler {
	return &AuthorizationDetailTypeHandler{store: store}
}

type authorizationDetailTypeRequest struct {
	TypeName         string   `json:"type_name"`
	Description      string   `json:"description"`
	JSONSchema       *string  `json:"json_schema"`
	AllowedActions   []string `json:"allowed_actions"`
	AllowedLocations []string `json:"allowed_locations"`
}

type authorizationDetailTypeResponse struct {
	ID               string   `json:"id"`
	TenantID         string   `json:"tenant_id"`
	TypeName         string   `json:"type_name"`
	Description      string   `json:"description"`
	JSONSchema       *string  `json:"json_schema,omitempty"`
	AllowedActions   []string `json:"allowed_actions"`
	AllowedLocations []string `json:"allowed_locations"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

func toAuthorizationDetailTypeResponse(adt *model.AuthorizationDetailType) *authorizationDetailTypeResponse {
	desc := ""
	if adt.Description != nil {
		desc = *adt.Description
	}
	return &authorizationDetailTypeResponse{
		ID:               adt.ID.String(),
		TenantID:         adt.TenantID.String(),
		TypeName:         adt.TypeName,
		Description:      desc,
		JSONSchema:       adt.JSONSchema,
		AllowedActions:   []string(adt.AllowedActions),
		AllowedLocations: []string(adt.AllowedLocations),
		CreatedAt:        adt.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:        adt.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// HandleList は GET /management/v1/tenants/:tenant_id/authorization-detail-types を処理する。
func (h *AuthorizationDetailTypeHandler) HandleList(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		return badRequest(c, "invalid tenant_id")
	}

	types, err := h.store.ListByTenantID(c.Request().Context(), tenantID)
	if err != nil {
		return serverError(c)
	}

	resp := make([]*authorizationDetailTypeResponse, 0, len(types))
	for _, t := range types {
		resp = append(resp, toAuthorizationDetailTypeResponse(&t))
	}

	return c.JSON(http.StatusOK, resp)
}

// HandleCreate は POST /management/v1/tenants/:tenant_id/authorization-detail-types を処理する。
func (h *AuthorizationDetailTypeHandler) HandleCreate(c echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("tenant_id"))
	if err != nil {
		return badRequest(c, "invalid tenant_id")
	}

	var req authorizationDetailTypeRequest
	if err := c.Bind(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	if req.TypeName == "" {
		return badRequest(c, "type_name is required")
	}

	if req.AllowedActions == nil {
		req.AllowedActions = []string{}
	}
	if req.AllowedLocations == nil {
		req.AllowedLocations = []string{}
	}

	var descPtr *string
	if req.Description != "" {
		descPtr = &req.Description
	}

	adt := &model.AuthorizationDetailType{
		TenantID:         tenantID,
		TypeName:         req.TypeName,
		Description:      descPtr,
		JSONSchema:       req.JSONSchema,
		AllowedActions:   model.StringSlice(req.AllowedActions),
		AllowedLocations: model.StringSlice(req.AllowedLocations),
	}

	if err := h.store.Create(c.Request().Context(), adt); err != nil {
		return serverError(c)
	}

	return c.JSON(http.StatusCreated, toAuthorizationDetailTypeResponse(adt))
}

// HandleGet は GET /management/v1/authorization-detail-types/:id を処理する。
func (h *AuthorizationDetailTypeHandler) HandleGet(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return badRequest(c, "invalid id")
	}

	adt, err := h.store.FindByID(c.Request().Context(), id)
	if err != nil {
		return serverError(c)
	}
	if adt == nil {
		return notFound(c, "authorization detail type not found")
	}

	return c.JSON(http.StatusOK, toAuthorizationDetailTypeResponse(adt))
}

// HandleUpdate は PUT /management/v1/authorization-detail-types/:id を処理する。
func (h *AuthorizationDetailTypeHandler) HandleUpdate(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return badRequest(c, "invalid id")
	}

	var req authorizationDetailTypeRequest
	if err := c.Bind(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	ctx := c.Request().Context()
	adt, err := h.store.FindByID(ctx, id)
	if err != nil {
		return serverError(c)
	}
	if adt == nil {
		return notFound(c, "authorization detail type not found")
	}

	if req.TypeName != "" {
		adt.TypeName = req.TypeName
	}
	if req.Description != "" {
		adt.Description = &req.Description
	}
	adt.JSONSchema = req.JSONSchema
	if req.AllowedActions != nil {
		adt.AllowedActions = model.StringSlice(req.AllowedActions)
	}
	if req.AllowedLocations != nil {
		adt.AllowedLocations = model.StringSlice(req.AllowedLocations)
	}

	if err := h.store.Update(ctx, adt); err != nil {
		return serverError(c)
	}

	return c.JSON(http.StatusOK, toAuthorizationDetailTypeResponse(adt))
}

// HandleDelete は DELETE /management/v1/authorization-detail-types/:id を処理する。
func (h *AuthorizationDetailTypeHandler) HandleDelete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return badRequest(c, "invalid id")
	}

	ctx := c.Request().Context()
	adt, err := h.store.FindByID(ctx, id)
	if err != nil {
		return serverError(c)
	}
	if adt == nil {
		return notFound(c, "authorization detail type not found")
	}

	if err := h.store.Delete(ctx, id); err != nil {
		return serverError(c)
	}

	return c.NoContent(http.StatusNoContent)
}
