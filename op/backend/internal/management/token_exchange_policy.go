package management

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// TokenExchangePolicyStore は Token Exchange ポリシーの永続化操作を定義する。
type TokenExchangePolicyStore interface {
	FindByClientID(ctx context.Context, clientID uuid.UUID) (*model.TokenExchangePolicy, error)
	Create(ctx context.Context, policy *model.TokenExchangePolicy) error
	Update(ctx context.Context, policy *model.TokenExchangePolicy) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// TokenExchangePolicyHandler は Token Exchange ポリシーの管理 API ハンドラ。
type TokenExchangePolicyHandler struct {
	store TokenExchangePolicyStore
}

// NewTokenExchangePolicyHandler は TokenExchangePolicyHandler を生成する。
func NewTokenExchangePolicyHandler(store TokenExchangePolicyStore) *TokenExchangePolicyHandler {
	return &TokenExchangePolicyHandler{store: store}
}

type tokenExchangePolicyRequest struct {
	AllowedSubjectTokenTypes   []string `json:"allowed_subject_token_types"`
	AllowedRequestedTokenTypes []string `json:"allowed_requested_token_types"`
	AllowedAudiences           []string `json:"allowed_audiences"`
	AllowImpersonation         bool     `json:"allow_impersonation"`
	AllowDelegation            bool     `json:"allow_delegation"`
}

type tokenExchangePolicyResponse struct {
	ID                         string   `json:"id"`
	ClientID                   string   `json:"client_id"`
	AllowedSubjectTokenTypes   []string `json:"allowed_subject_token_types"`
	AllowedRequestedTokenTypes []string `json:"allowed_requested_token_types"`
	AllowedAudiences           []string `json:"allowed_audiences"`
	AllowImpersonation         bool     `json:"allow_impersonation"`
	AllowDelegation            bool     `json:"allow_delegation"`
	CreatedAt                  string   `json:"created_at"`
	UpdatedAt                  string   `json:"updated_at"`
}

func toTokenExchangePolicyResponse(p *model.TokenExchangePolicy) *tokenExchangePolicyResponse {
	return &tokenExchangePolicyResponse{
		ID:                         p.ID.String(),
		ClientID:                   p.ClientID.String(),
		AllowedSubjectTokenTypes:   []string(p.AllowedSubjectTokenTypes),
		AllowedRequestedTokenTypes: []string(p.AllowedRequestedTokenTypes),
		AllowedAudiences:           []string(p.AllowedAudiences),
		AllowImpersonation:         p.AllowImpersonation,
		AllowDelegation:            p.AllowDelegation,
		CreatedAt:                  p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:                  p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// HandleGet は GET /management/v1/clients/:id/token-exchange-policy を処理する。
func (h *TokenExchangePolicyHandler) HandleGet(c echo.Context) error {
	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return badRequest(c, "invalid client id")
	}

	policy, err := h.store.FindByClientID(c.Request().Context(), clientID)
	if err != nil {
		return serverError(c)
	}
	if policy == nil {
		return notFound(c, "token exchange policy not found")
	}

	return c.JSON(http.StatusOK, toTokenExchangePolicyResponse(policy))
}

// HandleCreateOrUpdate は PUT /management/v1/clients/:id/token-exchange-policy を処理する（upsert）。
func (h *TokenExchangePolicyHandler) HandleCreateOrUpdate(c echo.Context) error {
	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return badRequest(c, "invalid client id")
	}

	var req tokenExchangePolicyRequest
	if err := c.Bind(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	// デフォルト値設定
	if req.AllowedSubjectTokenTypes == nil {
		req.AllowedSubjectTokenTypes = []string{"urn:ietf:params:oauth:token-type:access_token"}
	}
	if req.AllowedRequestedTokenTypes == nil {
		req.AllowedRequestedTokenTypes = []string{"urn:ietf:params:oauth:token-type:access_token"}
	}
	if req.AllowedAudiences == nil {
		req.AllowedAudiences = []string{}
	}

	ctx := c.Request().Context()
	existing, err := h.store.FindByClientID(ctx, clientID)
	if err != nil {
		return serverError(c)
	}

	if existing != nil {
		// Update
		existing.AllowedSubjectTokenTypes = model.StringSlice(req.AllowedSubjectTokenTypes)
		existing.AllowedRequestedTokenTypes = model.StringSlice(req.AllowedRequestedTokenTypes)
		existing.AllowedAudiences = model.StringSlice(req.AllowedAudiences)
		existing.AllowImpersonation = req.AllowImpersonation
		existing.AllowDelegation = req.AllowDelegation

		if err := h.store.Update(ctx, existing); err != nil {
			return serverError(c)
		}
		return c.JSON(http.StatusOK, toTokenExchangePolicyResponse(existing))
	}

	// Create
	policy := &model.TokenExchangePolicy{
		ClientID:                   clientID,
		AllowedSubjectTokenTypes:   model.StringSlice(req.AllowedSubjectTokenTypes),
		AllowedRequestedTokenTypes: model.StringSlice(req.AllowedRequestedTokenTypes),
		AllowedAudiences:           model.StringSlice(req.AllowedAudiences),
		AllowImpersonation:         req.AllowImpersonation,
		AllowDelegation:            req.AllowDelegation,
	}
	if err := h.store.Create(ctx, policy); err != nil {
		return serverError(c)
	}

	return c.JSON(http.StatusCreated, toTokenExchangePolicyResponse(policy))
}

// HandleDelete は DELETE /management/v1/clients/:id/token-exchange-policy を処理する。
func (h *TokenExchangePolicyHandler) HandleDelete(c echo.Context) error {
	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return badRequest(c, "invalid client id")
	}

	ctx := c.Request().Context()
	policy, err := h.store.FindByClientID(ctx, clientID)
	if err != nil {
		return serverError(c)
	}
	if policy == nil {
		return notFound(c, "token exchange policy not found")
	}

	if err := h.store.Delete(ctx, policy.ID); err != nil {
		return serverError(c)
	}

	return c.NoContent(http.StatusNoContent)
}
