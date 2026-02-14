package oidc

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type JWKSHandler struct {
	keySetProvider KeySetProvider
}

func NewJWKSHandler(keySetProvider KeySetProvider) *JWKSHandler {
	return &JWKSHandler{keySetProvider: keySetProvider}
}

// Handle は GET /jwks を処理する (RFC 7517 Section 5)
func (h *JWKSHandler) Handle(c echo.Context) error {
	set, err := h.keySetProvider.GetJWKSet(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	c.Response().Header().Set("Cache-Control", "public, max-age=3600")
	return c.JSON(http.StatusOK, set)
}
