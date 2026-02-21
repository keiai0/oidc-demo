package management

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

type redirectURIResponse struct {
	ID        string `json:"id"`
	URI       string `json:"uri"`
	CreatedAt string `json:"created_at"`
}

// RedirectURIHandler はリダイレクト URI 管理エンドポイントを処理する。
type RedirectURIHandler struct {
	redirectURIStore RedirectURIStore
	clientStore      ClientStore
}

// NewRedirectURIHandler は RedirectURIHandler を生成する。
func NewRedirectURIHandler(redirectURIStore RedirectURIStore, clientStore ClientStore) *RedirectURIHandler {
	return &RedirectURIHandler{
		redirectURIStore: redirectURIStore,
		clientStore:      clientStore,
	}
}

// HandleList は GET /management/v1/clients/:id/redirect-uris を処理する。
func (h *RedirectURIHandler) HandleList(c echo.Context) error {
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

	uris, err := h.redirectURIStore.ListByClientID(ctx, clientDBID)
	if err != nil {
		c.Logger().Errorf("failed to list redirect URIs: %v", err)
		return serverError(c)
	}

	data := make([]redirectURIResponse, len(uris))
	for i, u := range uris {
		data[i] = redirectURIResponse{
			ID:        u.ID.String(),
			URI:       u.URI,
			CreatedAt: u.CreatedAt.Format(time.RFC3339),
		}
	}

	return c.JSON(http.StatusOK, data)
}

type createRedirectURIRequest struct {
	URI string `json:"uri"`
}

// HandleCreate は POST /management/v1/clients/:id/redirect-uris を処理する。
func (h *RedirectURIHandler) HandleCreate(c echo.Context) error {
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

	var req createRedirectURIRequest
	if err := c.Bind(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	if err := validateRedirectURI(req.URI); err != nil {
		return badRequest(c, err.Error())
	}

	uri := &model.RedirectURI{
		ClientDBID: clientDBID,
		URI:        req.URI,
	}

	if err := h.redirectURIStore.Create(ctx, uri); err != nil {
		c.Logger().Errorf("failed to create redirect URI: %v", err)
		return serverError(c)
	}

	return c.JSON(http.StatusCreated, redirectURIResponse{
		ID:        uri.ID.String(),
		URI:       uri.URI,
		CreatedAt: uri.CreatedAt.Format(time.RFC3339),
	})
}

// HandleDelete は DELETE /management/v1/clients/:id/redirect-uris/:uri_id を処理する。
func (h *RedirectURIHandler) HandleDelete(c echo.Context) error {
	ctx := c.Request().Context()

	clientDBID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return badRequest(c, "invalid client id format")
	}

	uriID, err := uuid.Parse(c.Param("uri_id"))
	if err != nil {
		return badRequest(c, "invalid uri_id format")
	}

	// URI が指定クライアントに属するか確認
	uri, err := h.redirectURIStore.FindByID(ctx, uriID)
	if err != nil {
		c.Logger().Errorf("failed to find redirect URI: %v", err)
		return serverError(c)
	}
	if uri == nil {
		return notFound(c, "redirect URI not found")
	}
	if uri.ClientDBID != clientDBID {
		return notFound(c, "redirect URI not found for this client")
	}

	if err := h.redirectURIStore.Delete(ctx, uriID); err != nil {
		c.Logger().Errorf("failed to delete redirect URI: %v", err)
		return serverError(c)
	}

	return c.NoContent(http.StatusNoContent)
}
