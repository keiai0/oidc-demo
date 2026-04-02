package management

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// InitialAccessTokenStore は IAT の永続化操作を定義する。
type InitialAccessTokenStore interface {
	// Create は新しい IAT を永続化する。
	Create(ctx context.Context, iat *model.InitialAccessToken) error
	// FindByID は UUID で IAT を検索する。見つからない場合は (nil, nil)。
	FindByID(ctx context.Context, id uuid.UUID) (*model.InitialAccessToken, error)
	// ListByTenantID はテナントに属する IAT をページネーション付きで返す。
	ListByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]model.InitialAccessToken, int64, error)
	// Revoke は IAT を無効化する。
	Revoke(ctx context.Context, id uuid.UUID) error
}

// SHA256HexFunc は文字列の SHA256 ハッシュを hex 文字列で返す。
type SHA256HexFunc func(s string) string

// InitialAccessTokenHandler は Initial Access Token の管理 API ハンドラ。
type InitialAccessTokenHandler struct {
	store       InitialAccessTokenStore
	tenantStore TenantStore
	sha256Hex   SHA256HexFunc
}

// NewInitialAccessTokenHandler は InitialAccessTokenHandler を生成する。
func NewInitialAccessTokenHandler(store InitialAccessTokenStore, tenantStore TenantStore, sha256Hex SHA256HexFunc) *InitialAccessTokenHandler {
	return &InitialAccessTokenHandler{
		store:       store,
		tenantStore: tenantStore,
		sha256Hex:   sha256Hex,
	}
}

type createIATRequest struct {
	MaxRegistrations int    `json:"max_registrations"`
	ExpiresIn        int    `json:"expires_in"` // 秒数（デフォルト: 3600）
}

type iatResponse struct {
	ID               string  `json:"id"`
	TenantID         string  `json:"tenant_id"`
	MaxRegistrations int     `json:"max_registrations"`
	UsedCount        int     `json:"used_count"`
	ExpiresAt        string  `json:"expires_at"`
	RevokedAt        *string `json:"revoked_at,omitempty"`
	CreatedAt        string  `json:"created_at"`
}

type iatCreateResponse struct {
	iatResponse
	Token string `json:"token"` // 平文トークン（発行時のみ）
}

func toIATResponse(iat *model.InitialAccessToken) iatResponse {
	resp := iatResponse{
		ID:               iat.ID.String(),
		TenantID:         iat.TenantID.String(),
		MaxRegistrations: iat.MaxRegistrations,
		UsedCount:        iat.UsedCount,
		ExpiresAt:        iat.ExpiresAt.Format(time.RFC3339),
		CreatedAt:        iat.CreatedAt.Format(time.RFC3339),
	}
	if iat.RevokedAt != nil {
		s := iat.RevokedAt.Format(time.RFC3339)
		resp.RevokedAt = &s
	}
	return resp
}

// HandleCreate は POST /management/v1/tenants/:tenant_id/initial-access-tokens を処理する。
func (h *InitialAccessTokenHandler) HandleCreate(c echo.Context) error {
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

	var req createIATRequest
	if err := c.Bind(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	// デフォルト有効期限: 1時間
	expiresIn := req.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}

	// トークン生成 (32バイト = 64文字 hex)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		c.Logger().Errorf("failed to generate IAT: %v", err)
		return serverError(c)
	}
	token := hex.EncodeToString(tokenBytes)
	tokenHash := h.sha256Hex(token)

	iat := &model.InitialAccessToken{
		TokenHash:        tokenHash,
		TenantID:         tenantID,
		MaxRegistrations: req.MaxRegistrations,
		ExpiresAt:        time.Now().Add(time.Duration(expiresIn) * time.Second),
	}

	if err := h.store.Create(ctx, iat); err != nil {
		c.Logger().Errorf("failed to create IAT: %v", err)
		return serverError(c)
	}

	resp := iatCreateResponse{
		iatResponse: toIATResponse(iat),
		Token:       token,
	}

	return c.JSON(http.StatusCreated, resp)
}

// HandleList は GET /management/v1/tenants/:tenant_id/initial-access-tokens を処理する。
func (h *InitialAccessTokenHandler) HandleList(c echo.Context) error {
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
	tokens, total, err := h.store.ListByTenantID(ctx, tenantID, p.Limit, p.Offset)
	if err != nil {
		c.Logger().Errorf("failed to list IATs: %v", err)
		return serverError(c)
	}

	data := make([]iatResponse, len(tokens))
	for i, t := range tokens {
		data[i] = toIATResponse(&t)
	}

	return c.JSON(http.StatusOK, ListResponse[iatResponse]{
		Data:       data,
		TotalCount: total,
	})
}

// HandleRevoke は DELETE /management/v1/initial-access-tokens/:id を処理する。
func (h *InitialAccessTokenHandler) HandleRevoke(c echo.Context) error {
	ctx := c.Request().Context()

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return badRequest(c, "invalid id format")
	}

	iat, err := h.store.FindByID(ctx, id)
	if err != nil {
		c.Logger().Errorf("failed to find IAT: %v", err)
		return serverError(c)
	}
	if iat == nil {
		return notFound(c, "initial access token not found")
	}

	if err := h.store.Revoke(ctx, id); err != nil {
		c.Logger().Errorf("failed to revoke IAT: %v", err)
		return serverError(c)
	}

	return c.NoContent(http.StatusNoContent)
}

