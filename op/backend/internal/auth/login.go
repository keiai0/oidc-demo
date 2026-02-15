package auth

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

type LoginHandler struct {
	authSvc  *AuthService
	isSecure bool
}

func NewLoginHandler(authSvc *AuthService, isSecure bool) *LoginHandler {
	return &LoginHandler{authSvc: authSvc, isSecure: isSecure}
}

type loginRequest struct {
	TenantCode string `json:"tenant_code" validate:"required"`
	LoginID    string `json:"login_id" validate:"required"`
	Password   string `json:"password" validate:"required"`
}

// Handle は POST /internal/login を処理する
func (h *LoginHandler) Handle(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
	}

	if req.TenantCode == "" || req.LoginID == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "tenant_code, login_id, and password are required"})
	}

	input := &model.LoginInput{
		TenantCode: req.TenantCode,
		LoginID:    req.LoginID,
		Password:   req.Password,
		IPAddress:  c.RealIP(),
		UserAgent:  c.Request().UserAgent(),
	}

	output, err := h.authSvc.Login(c.Request().Context(), input)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid_credentials"})
		}
		c.Logger().Errorf("login error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	// セッションクッキーを設定
	cookie := &http.Cookie{
		Name:     "op_session",
		Value:    output.SessionID.String(),
		Path:     "/",
		HttpOnly: true,
		Secure:   h.isSecure,
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"session_id": output.SessionID.String(),
		"user": map[string]interface{}{
			"id":    output.User.ID.String(),
			"name":  output.User.Name,
			"email": output.User.Email,
		},
	})
}
