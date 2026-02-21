package management

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// IncidentHandler はトークン/セッションの一括失効を行うインシデント対応エンドポイントを処理する。
type IncidentHandler struct {
	sessionRevoker      SessionRevoker
	accessTokenRevoker  AccessTokenRevoker
	refreshTokenRevoker RefreshTokenRevoker
}

// NewIncidentHandler は IncidentHandler を生成する。
func NewIncidentHandler(
	sessionRevoker SessionRevoker,
	accessTokenRevoker AccessTokenRevoker,
	refreshTokenRevoker RefreshTokenRevoker,
) *IncidentHandler {
	return &IncidentHandler{
		sessionRevoker:      sessionRevoker,
		accessTokenRevoker:  accessTokenRevoker,
		refreshTokenRevoker: refreshTokenRevoker,
	}
}

type revokeTenantRequest struct {
	TenantID string `json:"tenant_id"`
}

type revokeUserRequest struct {
	UserID string `json:"user_id"`
}

type revokeResponse struct {
	Revoked struct {
		Sessions      int64 `json:"sessions"`
		AccessTokens  int64 `json:"access_tokens"`
		RefreshTokens int64 `json:"refresh_tokens"`
	} `json:"revoked"`
}

// HandleRevokeAll は POST /management/v1/incidents/revoke-all-tokens を処理する。
func (h *IncidentHandler) HandleRevokeAll(c echo.Context) error {
	ctx := c.Request().Context()

	sessions, err := h.sessionRevoker.RevokeAll(ctx)
	if err != nil {
		c.Logger().Errorf("failed to revoke all sessions: %v", err)
		return serverError(c)
	}
	accessTokens, err := h.accessTokenRevoker.RevokeAll(ctx)
	if err != nil {
		c.Logger().Errorf("failed to revoke all access tokens: %v", err)
		return serverError(c)
	}
	refreshTokens, err := h.refreshTokenRevoker.RevokeAll(ctx)
	if err != nil {
		c.Logger().Errorf("failed to revoke all refresh tokens: %v", err)
		return serverError(c)
	}

	var resp revokeResponse
	resp.Revoked.Sessions = sessions
	resp.Revoked.AccessTokens = accessTokens
	resp.Revoked.RefreshTokens = refreshTokens

	return c.JSON(http.StatusOK, resp)
}

// HandleRevokeTenant は POST /management/v1/incidents/revoke-tenant-tokens を処理する。
func (h *IncidentHandler) HandleRevokeTenant(c echo.Context) error {
	ctx := c.Request().Context()

	var req revokeTenantRequest
	if err := c.Bind(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return badRequest(c, "invalid tenant_id format")
	}

	sessions, err := h.sessionRevoker.RevokeByTenantID(ctx, tenantID)
	if err != nil {
		c.Logger().Errorf("failed to revoke tenant sessions: %v", err)
		return serverError(c)
	}
	accessTokens, err := h.accessTokenRevoker.RevokeByTenantID(ctx, tenantID)
	if err != nil {
		c.Logger().Errorf("failed to revoke tenant access tokens: %v", err)
		return serverError(c)
	}
	refreshTokens, err := h.refreshTokenRevoker.RevokeByTenantID(ctx, tenantID)
	if err != nil {
		c.Logger().Errorf("failed to revoke tenant refresh tokens: %v", err)
		return serverError(c)
	}

	var resp revokeResponse
	resp.Revoked.Sessions = sessions
	resp.Revoked.AccessTokens = accessTokens
	resp.Revoked.RefreshTokens = refreshTokens

	return c.JSON(http.StatusOK, resp)
}

// HandleRevokeUser は POST /management/v1/incidents/revoke-user-tokens を処理する。
func (h *IncidentHandler) HandleRevokeUser(c echo.Context) error {
	ctx := c.Request().Context()

	var req revokeUserRequest
	if err := c.Bind(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return badRequest(c, "invalid user_id format")
	}

	sessions, err := h.sessionRevoker.RevokeByUserID(ctx, userID)
	if err != nil {
		c.Logger().Errorf("failed to revoke user sessions: %v", err)
		return serverError(c)
	}
	accessTokens, err := h.accessTokenRevoker.RevokeByUserID(ctx, userID)
	if err != nil {
		c.Logger().Errorf("failed to revoke user access tokens: %v", err)
		return serverError(c)
	}
	refreshTokens, err := h.refreshTokenRevoker.RevokeByUserID(ctx, userID)
	if err != nil {
		c.Logger().Errorf("failed to revoke user refresh tokens: %v", err)
		return serverError(c)
	}

	var resp revokeResponse
	resp.Revoked.Sessions = sessions
	resp.Revoked.AccessTokens = accessTokens
	resp.Revoked.RefreshTokens = refreshTokens

	return c.JSON(http.StatusOK, resp)
}
