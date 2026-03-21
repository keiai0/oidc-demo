package scim

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// UserHandler は SCIM ユーザーエンドポイントを処理する。
type UserHandler struct {
	userStore    UserStore
	tenantFinder TenantFinder
	logger       *slog.Logger
	issuerBase   string
}

// NewUserHandler は UserHandler を生成する。
func NewUserHandler(
	userStore UserStore,
	tenantFinder TenantFinder,
	logger *slog.Logger,
	issuerBase string,
) *UserHandler {
	return &UserHandler{
		userStore:    userStore,
		tenantFinder: tenantFinder,
		logger:       logger,
		issuerBase:   issuerBase,
	}
}

func (h *UserHandler) usersBaseURL(tenantCode string) string {
	return h.issuerBase + "/" + tenantCode + "/scim/v2/Users"
}

// HandleList は GET /{tenant_code}/scim/v2/Users を処理する。
// RFC 7644 Section 3.4.2
func (h *UserHandler) HandleList(c echo.Context) error {
	ctx := c.Request().Context()
	tenantCode := c.Param("tenant_code")

	tenant, err := h.tenantFinder.FindByCode(ctx, tenantCode)
	if err != nil || tenant == nil {
		return scimErrorBadRequest(c, "unknown tenant")
	}

	// ページネーション: startIndex は 1-based (RFC 7644 Section 3.4.2.4)
	startIndex, _ := strconv.Atoi(c.QueryParam("startIndex"))
	if startIndex < 1 {
		startIndex = 1
	}
	count, _ := strconv.Atoi(c.QueryParam("count"))
	if count <= 0 || count > 100 {
		count = 100
	}
	offset := startIndex - 1

	// フィルタ
	filter, err := ParseFilter(c.QueryParam("filter"))
	if err != nil {
		return scimErrorBadRequest(c, fmt.Sprintf("invalid filter: %s", err))
	}

	// SCIM 属性名を DB カラム名に変換
	var filterAttr, filterValue string
	if filter != nil {
		filterValue = filter.Value
		switch filter.Attribute {
		case "userName":
			filterAttr = "login_id"
		case "email":
			filterAttr = "email"
		case "externalId":
			filterAttr = "external_id"
		case "active":
			filterAttr = "status"
			if filter.Value == "true" {
				filterValue = "active"
			} else {
				filterValue = "disabled"
			}
		}
	}

	users, total, err := h.userStore.ListByTenantIDWithFilter(ctx, tenant.ID, filterAttr, filterValue, offset, count)
	if err != nil {
		h.logger.ErrorContext(ctx, "scim list users error", "error", err)
		return scimErrorServer(c)
	}

	baseURL := h.usersBaseURL(tenantCode)
	resources := make([]SCIMUser, len(users))
	for i, u := range users {
		resources[i] = ToSCIMUser(&u, baseURL)
	}

	return c.JSON(http.StatusOK, SCIMListResponse{
		Schemas:      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		TotalResults: total,
		StartIndex:   startIndex,
		ItemsPerPage: len(resources),
		Resources:    resources,
	})
}

// HandleCreate は POST /{tenant_code}/scim/v2/Users を処理する。
// RFC 7644 Section 3.3
func (h *UserHandler) HandleCreate(c echo.Context) error {
	ctx := c.Request().Context()
	tenantCode := c.Param("tenant_code")

	tenant, err := h.tenantFinder.FindByCode(ctx, tenantCode)
	if err != nil || tenant == nil {
		return scimErrorBadRequest(c, "unknown tenant")
	}

	var req SCIMUserCreateRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return scimErrorBadRequest(c, "invalid request body")
	}

	if req.UserName == "" {
		return scimErrorBadRequest(c, "userName is required")
	}

	// userName の一意性チェック
	existing, err := h.userStore.FindByTenantAndLoginID(ctx, tenant.ID, req.UserName)
	if err != nil {
		h.logger.ErrorContext(ctx, "scim check uniqueness error", "error", err)
		return scimErrorServer(c)
	}
	if existing != nil {
		return scimErrorConflict(c, "userName already exists")
	}

	user := FromSCIMUserCreate(req, tenant.ID)
	if err := h.userStore.Create(ctx, user); err != nil {
		h.logger.ErrorContext(ctx, "scim create user error", "error", err)
		return scimErrorServer(c)
	}

	baseURL := h.usersBaseURL(tenantCode)
	scimUser := ToSCIMUser(user, baseURL)

	c.Response().Header().Set("Location", baseURL+"/"+user.ID.String())
	return c.JSON(http.StatusCreated, scimUser)
}

// HandleGet は GET /{tenant_code}/scim/v2/Users/:id を処理する。
// RFC 7644 Section 3.4.1
func (h *UserHandler) HandleGet(c echo.Context) error {
	ctx := c.Request().Context()
	tenantCode := c.Param("tenant_code")

	tenant, err := h.tenantFinder.FindByCode(ctx, tenantCode)
	if err != nil || tenant == nil {
		return scimErrorBadRequest(c, "unknown tenant")
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return scimErrorNotFound(c)
	}

	user, err := h.userStore.FindByID(ctx, id)
	if err != nil {
		h.logger.ErrorContext(ctx, "scim get user error", "error", err)
		return scimErrorServer(c)
	}
	if user == nil || user.TenantID != tenant.ID {
		return scimErrorNotFound(c)
	}

	baseURL := h.usersBaseURL(tenantCode)
	return c.JSON(http.StatusOK, ToSCIMUser(user, baseURL))
}

// HandlePatch は PATCH /{tenant_code}/scim/v2/Users/:id を処理する。
// RFC 7644 Section 3.5.2
func (h *UserHandler) HandlePatch(c echo.Context) error {
	ctx := c.Request().Context()
	tenantCode := c.Param("tenant_code")

	tenant, err := h.tenantFinder.FindByCode(ctx, tenantCode)
	if err != nil || tenant == nil {
		return scimErrorBadRequest(c, "unknown tenant")
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return scimErrorNotFound(c)
	}

	user, err := h.userStore.FindByID(ctx, id)
	if err != nil {
		h.logger.ErrorContext(ctx, "scim get user error", "error", err)
		return scimErrorServer(c)
	}
	if user == nil || user.TenantID != tenant.ID {
		return scimErrorNotFound(c)
	}

	var patchReq SCIMPatchRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&patchReq); err != nil {
		return scimErrorBadRequest(c, "invalid request body")
	}

	for _, op := range patchReq.Operations {
		if err := applyPatchOperation(user, op); err != nil {
			return scimErrorBadRequest(c, err.Error())
		}
	}

	if err := h.userStore.UpdateSCIM(ctx, user); err != nil {
		h.logger.ErrorContext(ctx, "scim update user error", "error", err)
		return scimErrorServer(c)
	}

	baseURL := h.usersBaseURL(tenantCode)
	return c.JSON(http.StatusOK, ToSCIMUser(user, baseURL))
}

// HandleDelete は DELETE /{tenant_code}/scim/v2/Users/:id を処理する。
// RFC 7644 Section 3.6 — ソフトデリート（status を disabled に変更）。
func (h *UserHandler) HandleDelete(c echo.Context) error {
	ctx := c.Request().Context()
	tenantCode := c.Param("tenant_code")

	tenant, err := h.tenantFinder.FindByCode(ctx, tenantCode)
	if err != nil || tenant == nil {
		return scimErrorBadRequest(c, "unknown tenant")
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return scimErrorNotFound(c)
	}

	user, err := h.userStore.FindByID(ctx, id)
	if err != nil {
		h.logger.ErrorContext(ctx, "scim get user error", "error", err)
		return scimErrorServer(c)
	}
	if user == nil || user.TenantID != tenant.ID {
		return scimErrorNotFound(c)
	}

	if err := h.userStore.Deactivate(ctx, id); err != nil {
		h.logger.ErrorContext(ctx, "scim deactivate user error", "error", err)
		return scimErrorServer(c)
	}

	return c.NoContent(http.StatusNoContent)
}

// applyPatchOperation は PATCH 操作をユーザーモデルに適用する。
func applyPatchOperation(user *model.User, op SCIMOperation) error {
	switch op.Op {
	case "replace":
		return applyReplace(user, op)
	case "add":
		return applyReplace(user, op) // add は replace と同等に処理
	case "remove":
		return applyRemove(user, op)
	default:
		return fmt.Errorf("unsupported operation: %s", op.Op)
	}
}

func applyReplace(user *model.User, op SCIMOperation) error {
	switch op.Path {
	case "userName":
		if v, ok := op.Value.(string); ok {
			user.LoginID = v
		}
	case "name.formatted":
		if v, ok := op.Value.(string); ok {
			user.Name = &v
		}
	case "emails[type eq \"work\"].value", "emails":
		// emails は配列または単一値の場合がある
		switch v := op.Value.(type) {
		case string:
			user.Email = v
		case []interface{}:
			if len(v) > 0 {
				if emailObj, ok := v[0].(map[string]interface{}); ok {
					if val, ok := emailObj["value"].(string); ok {
						user.Email = val
					}
				}
			}
		case map[string]interface{}:
			if val, ok := v["value"].(string); ok {
				user.Email = val
			}
		}
	case "active":
		switch v := op.Value.(type) {
		case bool:
			if v {
				user.Status = "active"
			} else {
				user.Status = "disabled"
			}
		case string:
			if v == "true" {
				user.Status = "active"
			} else {
				user.Status = "disabled"
			}
		}
	case "externalId":
		if v, ok := op.Value.(string); ok {
			user.ExternalID = &v
		}
	case "":
		// path なしの場合は value がオブジェクト
		if valueMap, ok := op.Value.(map[string]interface{}); ok {
			if v, ok := valueMap["active"]; ok {
				switch bv := v.(type) {
				case bool:
					if bv {
						user.Status = "active"
					} else {
						user.Status = "disabled"
					}
				}
			}
			if v, ok := valueMap["userName"].(string); ok {
				user.LoginID = v
			}
			if v, ok := valueMap["externalId"].(string); ok {
				user.ExternalID = &v
			}
		}
	default:
		return fmt.Errorf("unsupported path: %s", op.Path)
	}
	return nil
}

func applyRemove(user *model.User, op SCIMOperation) error {
	switch op.Path {
	case "name.formatted":
		user.Name = nil
	case "externalId":
		user.ExternalID = nil
	default:
		return fmt.Errorf("cannot remove path: %s", op.Path)
	}
	return nil
}
