package main

import (
	"context"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/isurugi-k/oidc-demo/op/backend/config"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/auth"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/crypto"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/database"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/jwt"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/management"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/oidc"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// マイグレーション実行
	if err := database.RunMigrations(cfg.DSN, "db/migrations"); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	log.Println("migrations completed successfully")

	// GORM 初期化
	db, err := database.NewDB(cfg.DSN)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	log.Println("database connected successfully")

	// Store 初期化
	tenantRepo := store.NewTenantRepository(db)
	clientRepo := store.NewClientRepository(db)
	userRepo := store.NewUserRepository(db)
	sessionRepo := store.NewSessionRepository(db)
	authCodeRepo := store.NewAuthorizationCodeRepository(db)
	accessTokenRepo := store.NewAccessTokenRepository(db)
	refreshTokenRepo := store.NewRefreshTokenRepository(db)
	idTokenRepo := store.NewIDTokenRepository(db)
	signKeyRepo := store.NewSignKeyRepository(db)
	redirectURIRepo := store.NewRedirectURIRepository(db)
	adminUserRepo := store.NewAdminUserRepository(db)
	adminSessionRepo := store.NewAdminSessionRepository(db)

	// JWT サービス初期化
	keySvc, err := jwt.NewKeyService(signKeyRepo, cfg.KeyEncryptionKey)
	if err != nil {
		log.Fatalf("failed to create key service: %v", err)
	}

	// 署名鍵がなければ自動生成
	if err := keySvc.EnsureSigningKey(context.Background()); err != nil {
		log.Fatalf("failed to ensure signing key: %v", err)
	}
	log.Println("signing key ensured")

	tokenSvc := jwt.NewTokenService(keySvc)

	// Auth サービス初期化
	authSvc := auth.NewAuthService(tenantRepo, userRepo, sessionRepo, crypto.VerifyPassword)

	// Auth ハンドラ初期化
	loginHandler := auth.NewLoginHandler(authSvc, cfg.IsSecure())
	meHandler := auth.NewMeHandler(authSvc, userRepo)

	// OIDC ハンドラ初期化
	jwksHandler := oidc.NewJWKSHandler(keySvc)
	discoveryHandler := oidc.NewDiscoveryHandler(cfg.BaseURL, tenantRepo)
	authorizeHandler := oidc.NewAuthorizeHandler(tenantRepo, clientRepo, authCodeRepo, authSvc, cfg.FrontendBaseURL)
	tokenHandler := oidc.NewTokenHandler(
		authCodeRepo, accessTokenRepo, refreshTokenRepo, idTokenRepo,
		clientRepo, tenantRepo, tokenSvc,
		crypto.VerifyPassword, crypto.VerifyCodeChallenge,
		jwt.ComputeATHash, jwt.SHA256Hex,
		cfg.BaseURL,
	)
	userInfoHandler := oidc.NewUserInfoHandler(tokenSvc, userRepo, accessTokenRepo)
	revokeHandler := oidc.NewRevokeHandler(clientRepo, accessTokenRepo, refreshTokenRepo, tokenSvc, crypto.VerifyPassword, jwt.SHA256Hex)

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// CORS (OP Frontend からの呼び出しを許可)
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{cfg.FrontendBaseURL},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderContentType, echo.HeaderAccept},
		AllowCredentials: true,
	}))

	// ヘルスチェック
	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// OIDC エンドポイント
	e.GET("/jwks", jwksHandler.Handle)
	e.GET("/:tenant_code/.well-known/openid-configuration", discoveryHandler.Handle)
	e.GET("/:tenant_code/authorize", authorizeHandler.Handle)
	e.POST("/:tenant_code/token", tokenHandler.Handle)
	e.GET("/:tenant_code/userinfo", userInfoHandler.Handle)
	e.POST("/:tenant_code/revoke", revokeHandler.Handle)

	// Internal API (OP Frontend 向け)
	e.POST("/internal/login", loginHandler.Handle)
	e.GET("/internal/me", meHandler.Handle)

	// Admin auth サービス初期化
	adminAuthSvc := management.NewAdminAuthService(adminUserRepo, adminSessionRepo, crypto.VerifyPassword)
	adminAuthHandler := management.NewAdminAuthHandler(adminAuthSvc, adminUserRepo, cfg.IsSecure())

	// Management auth エンドポイント (認証不要)
	e.POST("/management/v1/auth/login", adminAuthHandler.HandleLogin)
	e.GET("/management/v1/auth/me", adminAuthHandler.HandleMe)
	e.POST("/management/v1/auth/logout", adminAuthHandler.HandleLogout)

	// Management API (管理UI向け、セッション認証)
	mgmtGroup := e.Group("/management/v1", management.NewAuthMiddleware(adminAuthSvc))

	tenantMgmtHandler := management.NewTenantHandler(tenantRepo)
	mgmtGroup.GET("/tenants", tenantMgmtHandler.HandleList)
	mgmtGroup.POST("/tenants", tenantMgmtHandler.HandleCreate)
	mgmtGroup.GET("/tenants/:tenant_id", tenantMgmtHandler.HandleGet)
	mgmtGroup.PUT("/tenants/:tenant_id", tenantMgmtHandler.HandleUpdate)

	clientMgmtHandler := management.NewClientHandler(clientRepo, tenantRepo, crypto.HashPassword)
	mgmtGroup.GET("/tenants/:tenant_id/clients", clientMgmtHandler.HandleList)
	mgmtGroup.POST("/tenants/:tenant_id/clients", clientMgmtHandler.HandleCreate)
	mgmtGroup.GET("/clients/:id", clientMgmtHandler.HandleGet)
	mgmtGroup.PUT("/clients/:id", clientMgmtHandler.HandleUpdate)
	mgmtGroup.DELETE("/clients/:id", clientMgmtHandler.HandleDelete)
	mgmtGroup.PUT("/clients/:id/secret", clientMgmtHandler.HandleRotateSecret)

	redirectURIMgmtHandler := management.NewRedirectURIHandler(redirectURIRepo, clientRepo)
	mgmtGroup.GET("/clients/:id/redirect-uris", redirectURIMgmtHandler.HandleList)
	mgmtGroup.POST("/clients/:id/redirect-uris", redirectURIMgmtHandler.HandleCreate)
	mgmtGroup.DELETE("/clients/:id/redirect-uris/:uri_id", redirectURIMgmtHandler.HandleDelete)

	keyMgmtHandler := management.NewKeyHandler(signKeyRepo, keySvc)
	mgmtGroup.GET("/keys", keyMgmtHandler.HandleList)
	mgmtGroup.POST("/keys/rotate", keyMgmtHandler.HandleRotate)
	mgmtGroup.DELETE("/keys/:kid", keyMgmtHandler.HandleDeactivate)

	incidentHandler := management.NewIncidentHandler(sessionRepo, accessTokenRepo, refreshTokenRepo)
	mgmtGroup.POST("/incidents/revoke-all-tokens", incidentHandler.HandleRevokeAll)
	mgmtGroup.POST("/incidents/revoke-tenant-tokens", incidentHandler.HandleRevokeTenant)
	mgmtGroup.POST("/incidents/revoke-user-tokens", incidentHandler.HandleRevokeUser)

	e.Logger.Fatal(e.Start(":" + cfg.Port))
}
