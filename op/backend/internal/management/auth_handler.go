package management

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const adminCookieName = "op_admin_session"

// AdminAuthHandler は管理認証エンドポイントを処理する。
type AdminAuthHandler struct {
	authSvc    *AdminAuthService
	userFinder AdminUserFinder
	isSecure   bool
}

// NewAdminAuthHandler は AdminAuthHandler を生成する。
func NewAdminAuthHandler(authSvc *AdminAuthService, userFinder AdminUserFinder, isSecure bool) *AdminAuthHandler {
	return &AdminAuthHandler{
		authSvc:    authSvc,
		userFinder: userFinder,
		isSecure:   isSecure,
	}
}

type adminLoginRequest struct {
	LoginID  string `json:"login_id"`
	Password string `json:"password"`
}

// HandleLogin は POST /management/v1/auth/login を処理する。
func (h *AdminAuthHandler) HandleLogin(c echo.Context) error {
	var req adminLoginRequest
	if err := c.Bind(&req); err != nil {
		return badRequest(c, "invalid request body")
	}

	if req.LoginID == "" || req.Password == "" {
		return badRequest(c, "login_id and password are required")
	}

	session, user, err := h.authSvc.Login(
		c.Request().Context(),
		req.LoginID, req.Password,
		c.RealIP(), c.Request().UserAgent(),
	)
	if err != nil {
		if errors.Is(err, ErrAdminInvalidCredentials) {
			return errorJSON(c, http.StatusUnauthorized, "invalid_credentials", "login ID or password is incorrect")
		}
		c.Logger().Errorf("admin login error: %v", err)
		return serverError(c)
	}

	cookie := &http.Cookie{
		Name:     adminCookieName,
		Value:    session.ID.String(),
		Path:     "/",
		HttpOnly: true,
		Secure:   h.isSecure,
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user": map[string]interface{}{
			"id":       user.ID.String(),
			"login_id": user.LoginID,
			"name":     user.Name,
		},
	})
}

// HandleMe は GET /management/v1/auth/me を処理する。
func (h *AdminAuthHandler) HandleMe(c echo.Context) error {
	cookie, err := c.Cookie(adminCookieName)
	if err != nil {
		return errorJSON(c, http.StatusUnauthorized, "unauthorized", "no session")
	}

	sessionID, err := uuid.Parse(cookie.Value)
	if err != nil {
		return errorJSON(c, http.StatusUnauthorized, "unauthorized", "invalid session")
	}

	session, err := h.authSvc.ValidateSession(c.Request().Context(), sessionID)
	if err != nil {
		return errorJSON(c, http.StatusUnauthorized, "unauthorized", "session expired or invalid")
	}

	user, err := h.userFinder.FindByID(c.Request().Context(), session.AdminUserID)
	if err != nil || user == nil {
		return errorJSON(c, http.StatusUnauthorized, "unauthorized", "user not found")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user": map[string]interface{}{
			"id":       user.ID.String(),
			"login_id": user.LoginID,
			"name":     user.Name,
		},
	})
}

// HandleLogout は POST /management/v1/auth/logout を処理する。
func (h *AdminAuthHandler) HandleLogout(c echo.Context) error {
	cookie, err := c.Cookie(adminCookieName)
	if err == nil {
		sessionID, parseErr := uuid.Parse(cookie.Value)
		if parseErr == nil {
			_ = h.authSvc.RevokeSession(c.Request().Context(), sessionID)
		}
	}

	// Cookie をクリア
	clearCookie := &http.Cookie{
		Name:     adminCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.isSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	c.SetCookie(clearCookie)

	return c.NoContent(http.StatusNoContent)
}
