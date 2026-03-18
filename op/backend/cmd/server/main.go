package main

import (
	"context"
	"encoding/hex"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/isurugi-k/oidc-demo/op/backend/config"
	"github.com/go-webauthn/webauthn/protocol"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/audit"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/auth"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/crypto"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/database"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/health"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/jwt"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/metrics"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/management"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/oidc"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/store"
)

func main() {
	// 構造化ログ初期化（JSON 形式、stdout 出力）
	slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(slogLogger)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 監査ロガー初期化
	auditLog := audit.New(slogLogger)

	// マイグレーション実行
	if err := database.RunMigrations(cfg.DSN, "db/migrations"); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	slogLogger.Info("migrations completed successfully")

	// GORM 初期化
	db, err := database.NewDB(cfg.DSN)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	slogLogger.Info("database connected successfully")

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
	tenantClientRepo := store.NewTenantClientRepository(db)
	postLogoutRedirectURIRepo := store.NewPostLogoutRedirectURIRepository(db)
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
	slogLogger.Info("signing key ensured")

	tokenSvc := jwt.NewTokenService(keySvc)

	// MFA Store 初期化
	mfaConfigRepo := store.NewMfaConfigRepository(db)
	totpConfigRepo := store.NewTotpConfigRepository(db)

	// Auth サービス初期化
	authSvc := auth.NewAuthService(tenantRepo, userRepo, sessionRepo, userRepo, crypto.VerifyPassword, mfaConfigRepo)

	// パスワード関連の Store/Service 初期化
	passwordHistoryRepo := store.NewPasswordHistoryRepository(db)
	passwordCredentialRepo := store.NewPasswordCredentialRepository(db)
	passwordSvc := auth.NewPasswordService(userRepo, passwordHistoryRepo, passwordCredentialRepo, crypto.HashPassword, crypto.VerifyPassword)

	// パスワードリセット関連の初期化
	resetTokenRepo := store.NewPasswordResetTokenRepository(db)
	var emailSender auth.EmailSender
	if cfg.SMTPHost != "" {
		emailSender = auth.NewSMTPEmailSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPFrom, cfg.FrontendBaseURL)
		slogLogger.Info("email sender: SMTP", "host", cfg.SMTPHost, "port", cfg.SMTPPort)
	} else {
		emailSender = auth.NewStubEmailSender()
		slogLogger.Info("email sender: stub (set OP_SMTP_HOST to enable SMTP)")
	}
	passwordResetSvc := auth.NewPasswordResetService(tenantRepo, userRepo, resetTokenRepo, passwordSvc, userRepo, emailSender)

	// Consent 関連の初期化
	userConsentRepo := store.NewUserConsentRepository(db)

	// MFA TOTP サービス初期化
	// KeyEncryptionKey は hex 文字列なので、keySvc が既にデコード済みの鍵を持っている
	mfaEncKey, err := hex.DecodeString(cfg.KeyEncryptionKey)
	if err != nil {
		log.Fatalf("failed to decode key encryption key for MFA: %v", err)
	}
	mfaTOTPSvc := auth.NewMFATOTPService(
		mfaConfigRepo, mfaConfigRepo, totpConfigRepo, sessionRepo,
		crypto.Encrypt, crypto.Decrypt,
		mfaEncKey, "OIDC Demo",
	)

	// WebAuthn Store & Service 初期化
	webauthnCredRepo := store.NewWebAuthnCredentialRepository(db)

	webauthnLib, err := gowebauthn.New(&gowebauthn.Config{
		RPDisplayName: cfg.WebAuthnRPName,
		RPID:          cfg.WebAuthnRPID,
		RPOrigins:     cfg.WebAuthnRPOrigins,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			// パスキー（Discoverable Credential）として登録するために必須
			ResidentKey:             protocol.ResidentKeyRequirementRequired,
			UserVerification:        protocol.VerificationPreferred,
			// platform: ブラウザのパスワードマネージャー（Touch ID / Windows Hello）を使用
			AuthenticatorAttachment: protocol.Platform,
		},
	})
	if err != nil {
		log.Fatalf("failed to create WebAuthn instance: %v", err)
	}

	mfaWebAuthnSvc := auth.NewMFAWebAuthnService(
		webauthnLib, mfaConfigRepo, mfaConfigRepo,
		webauthnCredRepo, sessionRepo, sessionRepo,
	)

	// Auth ハンドラ初期化
	loginHandler := auth.NewLoginHandler(authSvc, auditLog, slogLogger, cfg.IsSecure())
	meHandler := auth.NewMeHandler(authSvc, userRepo)
	passwordChangeHandler := auth.NewPasswordChangeHandler(passwordSvc, authSvc, cfg.IsSecure())
	consentHandler := auth.NewConsentHandler(authSvc, userConsentRepo)
	mfaSetupHandler := auth.NewMFATOTPSetupHandler(mfaTOTPSvc, authSvc, userRepo)
	mfaVerifySetupHandler := auth.NewMFATOTPVerifySetupHandler(mfaTOTPSvc, authSvc)
	mfaVerifyHandler := auth.NewMFATOTPVerifyHandler(mfaTOTPSvc, authSvc, auditLog, slogLogger)

	// WebAuthn ハンドラ初期化
	webauthnRegBeginHandler := auth.NewMFAWebAuthnRegisterBeginHandler(mfaWebAuthnSvc, authSvc, userRepo)
	webauthnRegCompleteHandler := auth.NewMFAWebAuthnRegisterCompleteHandler(mfaWebAuthnSvc, authSvc)
	webauthnAuthBeginHandler := auth.NewMFAWebAuthnAuthBeginHandler(mfaWebAuthnSvc, authSvc)
	webauthnAuthCompleteHandler := auth.NewMFAWebAuthnAuthCompleteHandler(mfaWebAuthnSvc, authSvc, auditLog, slogLogger)
	webauthnCredsHandler := auth.NewMFAWebAuthnCredentialsHandler(mfaWebAuthnSvc, authSvc)
	resetRequestHandler := auth.NewPasswordResetRequestHandler(passwordResetSvc)
	resetHandler := auth.NewPasswordResetHandler(passwordResetSvc)

	// Phase 10: メールアドレス変更 関連の初期化
	emailChangeTokenRepo := store.NewEmailChangeTokenRepository(db)
	emailChangeSvc := auth.NewEmailChangeService(emailChangeTokenRepo, userRepo, emailSender)
	emailChangeHandler := auth.NewEmailChangeHandler(emailChangeSvc, authSvc)

	// Phase 10: バックアップコード 関連の初期化
	backupCodeRepo := store.NewBackupCodeRepository(db)
	backupCodeSvc := auth.NewBackupCodeService(backupCodeRepo, sessionRepo, crypto.HashPassword, crypto.VerifyPassword)
	backupCodeHandler := auth.NewMFABackupCodeHandler(backupCodeSvc, authSvc)

	// Phase 10: TOTP 無効化ハンドラ
	mfaTOTPDisableHandler := auth.NewMFATOTPDisableHandler(mfaTOTPSvc, authSvc, userRepo, crypto.VerifyPassword)

	// Phase 10: セッション一覧・失効ハンドラ
	sessionListHandler := auth.NewSessionListHandler(sessionRepo, sessionRepo, sessionRepo, authSvc)

	// パスキーログインサービス & ハンドラ初期化
	passkeyLoginSvc := auth.NewPasskeyLoginService(
		webauthnLib, mfaConfigRepo, webauthnCredRepo,
		sessionRepo, userRepo, tenantRepo,
	)
	passkeyLoginHandler := auth.NewPasskeyLoginHandler(passkeyLoginSvc, cfg.IsSecure())

	// PAR Store 初期化
	parRepo := store.NewPushedAuthorizationRequestRepository(db)

	// DPoP JTI Cache 初期化
	dpopJTIRepo := store.NewDPoPJTICacheRepository(db)

	// OIDC ハンドラ初期化
	jwksHandler := oidc.NewJWKSHandler(keySvc)
	discoveryHandler := oidc.NewDiscoveryHandler(cfg.BaseURL, tenantRepo)
	authorizeHandler := oidc.NewAuthorizeHandler(tenantRepo, clientRepo, tenantClientRepo, authCodeRepo, userConsentRepo, authSvc, parRepo, cfg.FrontendBaseURL)
	parHandler := oidc.NewPARHandler(clientRepo, tenantRepo, tenantClientRepo, parRepo, crypto.VerifyPassword)
	tokenHandler := oidc.NewTokenHandler(
		authCodeRepo, accessTokenRepo, refreshTokenRepo, idTokenRepo,
		clientRepo, tenantRepo, tenantClientRepo, userRepo, tokenSvc,
		crypto.VerifyPassword, crypto.VerifyCodeChallenge,
		jwt.ComputeATHash, jwt.SHA256Hex,
		dpopJTIRepo,
		auditLog, slogLogger,
		cfg.BaseURL,
	)
	userInfoHandler := oidc.NewUserInfoHandler(tokenSvc, userRepo, clientRepo, accessTokenRepo, dpopJTIRepo, tokenSvc, cfg.BaseURL)
	revokeHandler := oidc.NewRevokeHandler(clientRepo, accessTokenRepo, refreshTokenRepo, tokenSvc, crypto.VerifyPassword, jwt.SHA256Hex, auditLog)
	introspectHandler := oidc.NewIntrospectHandler(clientRepo, accessTokenRepo, refreshTokenRepo, tokenSvc, userRepo, crypto.VerifyPassword, jwt.SHA256Hex)

	// SLO (Single Logout) ハンドラ初期化
	backChannelClient := &http.Client{Timeout: 10 * time.Second}
	logoutHandler := oidc.NewLogoutHandler(
		tenantRepo, clientRepo, tenantClientRepo,
		clientRepo, postLogoutRedirectURIRepo,
		tokenSvc, tokenSvc,
		sessionRepo, accessTokenRepo, refreshTokenRepo,
		cfg.BaseURL, cfg.FrontendBaseURL,
		backChannelClient, cfg.IsSecure(),
	)
	internalLogoutHandler := auth.NewInternalLogoutHandler(
		sessionRepo, accessTokenRepo, refreshTokenRepo, auditLog, cfg.IsSecure(),
	)

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(metrics.HTTPMetricsMiddleware())

	// セキュリティヘッダー（全レスポンス）
	e.Use(oidc.SecurityHeadersMiddleware(cfg.IsSecure()))

	// CORS: OP Frontend + RP オリジンを統合（/internal/* は AllowCredentials で Cookie 送信許可）
	allOrigins := append([]string{cfg.FrontendBaseURL}, cfg.AllowedRPOrigins...)
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     allOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "DPoP"},
		AllowCredentials: true,
	}))

	// ヘルスチェック
	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Liveness / Readiness Probe
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get sql.DB: %v", err)
	}
	livenessHandler := health.NewLivenessHandler()
	readinessHandler := health.NewReadinessHandler(sqlDB, keySvc)
	e.GET("/health", livenessHandler.Handle)
	e.GET("/ready", readinessHandler.Handle)

	// Prometheus メトリクス
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// OIDC エンドポイント
	e.GET("/jwks", jwksHandler.Handle)
	e.GET("/:tenant_code/.well-known/openid-configuration", discoveryHandler.Handle)
	e.GET("/:tenant_code/authorize", authorizeHandler.Handle)
	e.POST("/:tenant_code/token", tokenHandler.Handle)
	e.GET("/:tenant_code/userinfo", userInfoHandler.Handle)
	e.POST("/:tenant_code/revoke", revokeHandler.Handle)
	e.POST("/:tenant_code/introspect", introspectHandler.Handle)
	e.POST("/:tenant_code/par", parHandler.Handle)
	e.GET("/:tenant_code/logout", logoutHandler.Handle)
	e.POST("/:tenant_code/logout", logoutHandler.Handle)

	// Internal API (OP Frontend 向け)
	loginRateLimiter := auth.NewRateLimiter(10, 1*time.Minute)
	e.POST("/internal/login", loginHandler.Handle, loginRateLimiter.Middleware())
	e.POST("/internal/logout", internalLogoutHandler.Handle)
	e.GET("/internal/me", meHandler.Handle)
	e.POST("/internal/password/change", passwordChangeHandler.Handle)
	e.POST("/internal/password/reset-request", resetRequestHandler.Handle)
	e.POST("/internal/password/reset", resetHandler.Handle)
	e.POST("/internal/consent", consentHandler.Handle)
	e.POST("/internal/passkey/login/begin", passkeyLoginHandler.HandleBegin)
	e.POST("/internal/passkey/login/complete", passkeyLoginHandler.HandleComplete)
	e.POST("/internal/mfa/totp/setup", mfaSetupHandler.Handle)
	e.POST("/internal/mfa/totp/verify-setup", mfaVerifySetupHandler.Handle)
	e.POST("/internal/mfa/totp/verify", mfaVerifyHandler.Handle)
	e.POST("/internal/mfa/webauthn/register/begin", webauthnRegBeginHandler.Handle)
	e.POST("/internal/mfa/webauthn/register/complete", webauthnRegCompleteHandler.Handle)
	e.POST("/internal/mfa/webauthn/authenticate/begin", webauthnAuthBeginHandler.Handle)
	e.POST("/internal/mfa/webauthn/authenticate/complete", webauthnAuthCompleteHandler.Handle)
	e.GET("/internal/mfa/webauthn/credentials", webauthnCredsHandler.HandleList)
	e.DELETE("/internal/mfa/webauthn/credentials/:id", webauthnCredsHandler.HandleDelete)

	// Phase 10: ユーザーセルフサービス
	e.POST("/internal/email/change-request", emailChangeHandler.HandleRequest)
	e.POST("/internal/email/verify", emailChangeHandler.HandleVerify)
	e.DELETE("/internal/mfa/totp", mfaTOTPDisableHandler.Handle)
	e.POST("/internal/mfa/backup-codes/generate", backupCodeHandler.HandleGenerate)
	e.POST("/internal/mfa/backup-codes/verify", backupCodeHandler.HandleVerify)
	e.GET("/internal/sessions", sessionListHandler.HandleList)
	e.DELETE("/internal/sessions/:id", sessionListHandler.HandleRevoke)

	// Admin auth サービス初期化
	adminAuthSvc := management.NewAdminAuthService(adminUserRepo, adminSessionRepo, crypto.VerifyPassword)
	adminAuthHandler := management.NewAdminAuthHandler(adminAuthSvc, adminUserRepo, auditLog, slogLogger, cfg.IsSecure())

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

	clientMgmtHandler := management.NewClientHandler(clientRepo, tenantRepo, tenantClientRepo, crypto.HashPassword)
	mgmtGroup.GET("/clients", clientMgmtHandler.HandleListAll)
	mgmtGroup.GET("/tenants/:tenant_id/clients", clientMgmtHandler.HandleList)
	mgmtGroup.POST("/tenants/:tenant_id/clients", clientMgmtHandler.HandleCreate)
	mgmtGroup.GET("/clients/:id", clientMgmtHandler.HandleGet)
	mgmtGroup.PUT("/clients/:id", clientMgmtHandler.HandleUpdate)
	mgmtGroup.DELETE("/clients/:id", clientMgmtHandler.HandleDelete)
	mgmtGroup.PUT("/clients/:id/secret", clientMgmtHandler.HandleRotateSecret)
	mgmtGroup.GET("/clients/:id/tenants", clientMgmtHandler.HandleListTenants)
	mgmtGroup.POST("/clients/:id/tenants", clientMgmtHandler.HandleAddTenant)
	mgmtGroup.DELETE("/clients/:id/tenants/:tenant_id", clientMgmtHandler.HandleRemoveTenant)

	redirectURIMgmtHandler := management.NewRedirectURIHandler(redirectURIRepo, clientRepo)
	mgmtGroup.GET("/clients/:id/redirect-uris", redirectURIMgmtHandler.HandleList)
	mgmtGroup.POST("/clients/:id/redirect-uris", redirectURIMgmtHandler.HandleCreate)
	mgmtGroup.DELETE("/clients/:id/redirect-uris/:uri_id", redirectURIMgmtHandler.HandleDelete)

	keyMgmtHandler := management.NewKeyHandler(signKeyRepo, keySvc)
	mgmtGroup.GET("/keys", keyMgmtHandler.HandleList)
	mgmtGroup.POST("/keys/rotate", keyMgmtHandler.HandleRotate)
	mgmtGroup.DELETE("/keys/:kid", keyMgmtHandler.HandleDeactivate)

	incidentHandler := management.NewIncidentHandler(sessionRepo, accessTokenRepo, refreshTokenRepo, userRepo)
	mgmtGroup.POST("/incidents/revoke-all-tokens", incidentHandler.HandleRevokeAll)
	mgmtGroup.POST("/incidents/revoke-tenant-tokens", incidentHandler.HandleRevokeTenant)
	mgmtGroup.POST("/incidents/revoke-user-tokens", incidentHandler.HandleRevokeUser)
	mgmtGroup.POST("/users/:user_id/unlock", incidentHandler.HandleUnlockUser)

	// 署名鍵自動ローテーション scheduler 起動
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	rotationScheduler := jwt.NewRotationScheduler(keySvc, auditLog, slogLogger, cfg.KeyRotationIntervalDays, cfg.KeyGracePeriodDays)
	go rotationScheduler.Run(ctx)

	// graceful shutdown: ctx がキャンセルされたらサーバーをシャットダウン
	go func() {
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := e.Shutdown(shutdownCtx); err != nil {
			slogLogger.Error("server shutdown error", "error", err)
		}
	}()

	if err := e.Start(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
