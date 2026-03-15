package oidc

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type UserInfoHandler struct {
	tokenValidator   TokenValidator
	userFinder       UserFinder
	accessTokenStore AccessTokenStore
	dpopJTIStore     DPoPJTIStore
	issuerBaseURL    string
}

func NewUserInfoHandler(
	tokenValidator TokenValidator,
	userFinder UserFinder,
	accessTokenStore AccessTokenStore,
	dpopJTIStore DPoPJTIStore,
	issuerBaseURL string,
) *UserInfoHandler {
	return &UserInfoHandler{
		tokenValidator:   tokenValidator,
		userFinder:       userFinder,
		accessTokenStore: accessTokenStore,
		dpopJTIStore:     dpopJTIStore,
		issuerBaseURL:    issuerBaseURL,
	}
}

// Handle は GET /{tenant_code}/userinfo を処理する
// 仕様参照: OIDC Core 1.0 Section 5.3, RFC 9449 Section 7.1
func (h *UserInfoHandler) Handle(c echo.Context) error {
	// Bearer または DPoP トークン取得
	authHeader := c.Request().Header.Get("Authorization")
	var tokenString string
	var isDPoP bool

	if strings.HasPrefix(authHeader, "DPoP ") {
		tokenString = strings.TrimPrefix(authHeader, "DPoP ")
		isDPoP = true
	} else if strings.HasPrefix(authHeader, "Bearer ") {
		tokenString = strings.TrimPrefix(authHeader, "Bearer ")
	} else {
		c.Response().Header().Set("WWW-Authenticate", `Bearer`)
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
	}

	ctx := c.Request().Context()

	// アクセストークン検証 (JWT 署名検証 + 有効期限)
	result, err := h.tokenValidator.ValidateAccessToken(ctx, tokenString)
	if err != nil {
		c.Response().Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
	}

	// DB でアクセストークンの失効チェック
	dbToken, err := h.accessTokenStore.FindByJTI(ctx, result.JTI)
	if err != nil || dbToken == nil || dbToken.RevokedAt != nil {
		c.Response().Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
	}

	// DPoP 検証 (RFC 9449 Section 7.1)
	if result.DPoPJKT != nil {
		// トークンが DPoP bound なら DPoP スキームが必須
		if !isDPoP {
			c.Response().Header().Set("WWW-Authenticate", `DPoP error="invalid_token", error_description="DPoP-bound token requires DPoP scheme"`)
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
		}

		// DPoP proof 検証
		dpopProof := c.Request().Header.Get("DPoP")
		if dpopProof == "" {
			c.Response().Header().Set("WWW-Authenticate", `DPoP error="invalid_token"`)
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
		}

		tenantCode := c.Param("tenant_code")
		httpURL := h.issuerBaseURL + "/" + tenantCode + "/userinfo"

		dpopResult, err := VerifyDPoPProof(ctx, dpopProof, "GET", httpURL, h.dpopJTIStore)
		if err != nil {
			c.Response().Header().Set("WWW-Authenticate", `DPoP error="invalid_token"`)
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
		}

		// cnf.jkt と DPoP proof の JWK Thumbprint が一致するか検証
		if dpopResult.JKT != *result.DPoPJKT {
			c.Response().Header().Set("WWW-Authenticate", `DPoP error="invalid_token"`)
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
		}
	}

	// ユーザー情報取得
	user, err := h.userFinder.FindByID(ctx, result.Subject)
	if err != nil || user == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}

	// スコープに応じたクレーム構築
	claims := map[string]interface{}{
		"sub": user.ID.String(),
	}

	scopes := strings.Split(result.Scope, " ")

	if containsScope(scopes, "profile") {
		if user.Name != nil {
			claims["name"] = *user.Name
		}
		claims["updated_at"] = user.UpdatedAt.Unix()
	}

	if containsScope(scopes, "email") {
		claims["email"] = user.Email
		claims["email_verified"] = user.EmailVerified
	}

	return c.JSON(http.StatusOK, claims)
}
