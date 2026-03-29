package test

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/labstack/echo/v4"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/audit"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/auth"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/crypto"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/database"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/jwt"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/management"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/oidc"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/store"

	"gorm.io/gorm"
)

var (
	// serverURL はテストサーバーの URL（例: "http://127.0.0.1:12345"）
	serverURL string

	// testDB はフィクスチャ操作用の GORM インスタンス
	testDB *gorm.DB

	// httpClient はリダイレクトを追跡しない HTTP クライアント
	httpClient *http.Client

	// cachedPasswordHash は argon2id ハッシュのキャッシュ（テスト高速化用）
	cachedPasswordHash string

	// cachedSecretHash はクライアントシークレットの argon2id ハッシュキャッシュ
	cachedSecretHash string
)

const (
	// testPassword はテスト用の固定パスワード
	testPassword = "Test1234!"

	// testClientSecret はテスト用の固定クライアントシークレット
	testClientSecret = "test-client-secret-value"

	// testKeyEncryptionKey はテスト用の AES-256-GCM 鍵（64文字 hex = 32バイト）
	testKeyEncryptionKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

func TestMain(m *testing.M) {
	// テスト用 DSN（環境変数で上書き可能）
	dsn := os.Getenv("OP_TEST_DSN")
	if dsn == "" {
		// docker-compose のポートを自動検出
		port := os.Getenv("OP_TEST_DB_PORT")
		if port == "" {
			port = "5432"
		}
		dsn = fmt.Sprintf("postgres://op_user:op_password@localhost:%s/oidc_demo?search_path=op&sslmode=disable", port)
	}

	// DB リセット + マイグレーション
	if err := database.FreshMigrations(dsn, "../db/migrations"); err != nil {
		log.Fatalf("FreshMigrations failed: %v", err)
	}

	// GORM 接続
	db, err := database.NewDB(dsn)
	if err != nil {
		log.Fatalf("NewDB failed: %v", err)
	}
	testDB = db

	// argon2id ハッシュをキャッシュ（低速処理を1回だけ実行）
	cachedPasswordHash, err = crypto.HashPassword(testPassword)
	if err != nil {
		log.Fatalf("HashPassword failed: %v", err)
	}
	cachedSecretHash, err = crypto.HashPassword(testClientSecret)
	if err != nil {
		log.Fatalf("HashPassword for secret failed: %v", err)
	}

	// Echo サーバー + DI 組み立て
	server := setupServer(db)
	serverURL = server.URL

	// HTTP クライアント（リダイレクト追跡しない）
	httpClient = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 10 * time.Second,
	}

	code := m.Run()
	server.Close()
	os.Exit(code)
}

// setupServer は main.go と同じ DI 組み立てを行い、httptest.NewServer を返す。
func setupServer(db *gorm.DB) *httptest.Server {
	slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	slog.SetDefault(slogLogger)

	auditLog := audit.New(slogLogger)

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
	tenantClientRepo := store.NewTenantClientRepository(db)
	postLogoutRedirectURIRepo := store.NewPostLogoutRedirectURIRepository(db)
	adminUserRepo := store.NewAdminUserRepository(db)
	adminSessionRepo := store.NewAdminSessionRepository(db)

	// JWT サービス初期化
	keySvc, err := jwt.NewKeyService(signKeyRepo, testKeyEncryptionKey)
	if err != nil {
		log.Fatalf("NewKeyService failed: %v", err)
	}
	if err := keySvc.EnsureSigningKey(context.Background()); err != nil {
		log.Fatalf("EnsureSigningKey failed: %v", err)
	}
	tokenSvc := jwt.NewTokenService(keySvc)

	// MFA Store 初期化
	mfaConfigRepo := store.NewMfaConfigRepository(db)
	totpConfigRepo := store.NewTotpConfigRepository(db)

	// Auth サービス初期化
	authSvc := auth.NewAuthService(tenantRepo, userRepo, sessionRepo, userRepo, crypto.VerifyPassword, mfaConfigRepo)

	// パスワード関連
	passwordHistoryRepo := store.NewPasswordHistoryRepository(db)
	passwordCredentialRepo := store.NewPasswordCredentialRepository(db)
	passwordSvc := auth.NewPasswordService(userRepo, passwordHistoryRepo, passwordCredentialRepo, crypto.HashPassword, crypto.VerifyPassword)
	resetTokenRepo := store.NewPasswordResetTokenRepository(db)
	emailSender := auth.NewStubEmailSender()
	passwordResetSvc := auth.NewPasswordResetService(tenantRepo, userRepo, resetTokenRepo, passwordSvc, userRepo, emailSender)

	// Consent
	userConsentRepo := store.NewUserConsentRepository(db)

	// MFA TOTP
	mfaEncKey, err := hex.DecodeString(testKeyEncryptionKey)
	if err != nil {
		log.Fatalf("DecodeString failed: %v", err)
	}
	mfaTOTPSvc := auth.NewMFATOTPService(
		mfaConfigRepo, mfaConfigRepo, totpConfigRepo, sessionRepo,
		crypto.Encrypt, crypto.Decrypt,
		mfaEncKey, "OIDC Demo",
	)

	// WebAuthn
	webauthnCredRepo := store.NewWebAuthnCredentialRepository(db)
	webauthnLib, err := gowebauthn.New(&gowebauthn.Config{
		RPDisplayName: "OIDC Demo Test",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:3000"},
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:             protocol.ResidentKeyRequirementRequired,
			UserVerification:        protocol.VerificationPreferred,
			AuthenticatorAttachment: protocol.Platform,
		},
	})
	if err != nil {
		log.Fatalf("WebAuthn New failed: %v", err)
	}
	mfaWebAuthnSvc := auth.NewMFAWebAuthnService(
		webauthnLib, mfaConfigRepo, mfaConfigRepo,
		webauthnCredRepo, sessionRepo, sessionRepo,
	)

	// Auth ハンドラ
	loginHandler := auth.NewLoginHandler(authSvc, auditLog, slogLogger, false)
	meHandler := auth.NewMeHandler(authSvc, userRepo)
	passwordChangeHandler := auth.NewPasswordChangeHandler(passwordSvc, authSvc, false)
	consentHandler := auth.NewConsentHandler(authSvc, userConsentRepo)
	mfaSetupHandler := auth.NewMFATOTPSetupHandler(mfaTOTPSvc, authSvc, userRepo)
	mfaVerifySetupHandler := auth.NewMFATOTPVerifySetupHandler(mfaTOTPSvc, authSvc)
	mfaVerifyHandler := auth.NewMFATOTPVerifyHandler(mfaTOTPSvc, authSvc, auditLog, slogLogger)
	webauthnRegBeginHandler := auth.NewMFAWebAuthnRegisterBeginHandler(mfaWebAuthnSvc, authSvc, userRepo)
	webauthnRegCompleteHandler := auth.NewMFAWebAuthnRegisterCompleteHandler(mfaWebAuthnSvc, authSvc)
	webauthnAuthBeginHandler := auth.NewMFAWebAuthnAuthBeginHandler(mfaWebAuthnSvc, authSvc)
	webauthnAuthCompleteHandler := auth.NewMFAWebAuthnAuthCompleteHandler(mfaWebAuthnSvc, authSvc, auditLog, slogLogger)
	webauthnCredsHandler := auth.NewMFAWebAuthnCredentialsHandler(mfaWebAuthnSvc, authSvc)
	resetRequestHandler := auth.NewPasswordResetRequestHandler(passwordResetSvc)
	resetHandler := auth.NewPasswordResetHandler(passwordResetSvc)

	// Email Change
	emailChangeTokenRepo := store.NewEmailChangeTokenRepository(db)
	emailChangeSvc := auth.NewEmailChangeService(emailChangeTokenRepo, userRepo, emailSender)
	emailChangeHandler := auth.NewEmailChangeHandler(emailChangeSvc, authSvc)

	// Backup Code
	backupCodeRepo := store.NewBackupCodeRepository(db)
	backupCodeSvc := auth.NewBackupCodeService(backupCodeRepo, sessionRepo, crypto.HashPassword, crypto.VerifyPassword)
	backupCodeHandler := auth.NewMFABackupCodeHandler(backupCodeSvc, authSvc)

	// TOTP Disable
	mfaTOTPDisableHandler := auth.NewMFATOTPDisableHandler(mfaTOTPSvc, authSvc, userRepo, crypto.VerifyPassword)

	// Session List
	sessionListHandler := auth.NewSessionListHandler(sessionRepo, sessionRepo, sessionRepo, authSvc)

	// Passkey Login
	passkeyLoginSvc := auth.NewPasskeyLoginService(
		webauthnLib, mfaConfigRepo, webauthnCredRepo,
		sessionRepo, userRepo, tenantRepo,
	)
	passkeyLoginHandler := auth.NewPasskeyLoginHandler(passkeyLoginSvc, false)

	// PAR / DPoP / Device Auth
	parRepo := store.NewPushedAuthorizationRequestRepository(db)
	dpopJTIRepo := store.NewDPoPJTICacheRepository(db)
	deviceAuthRepo := store.NewDeviceAuthorizationRequestRepository(db)
	tokenExchangePolicyRepo := store.NewTokenExchangePolicyRepository(db)

	// Federation
	federationProviderRepo := store.NewFederationProviderRepository(db)
	externalIdPCredRepo := store.NewExternalIdPCredentialRepository(db)
	credentialRepo := store.NewCredentialRepository(db)
	federationHTTPClient := &http.Client{Timeout: 10 * time.Second}
	federationSvc := auth.NewFederationService(
		federationProviderRepo, externalIdPCredRepo, credentialRepo, userRepo,
		sessionRepo, tenantRepo, crypto.Decrypt, mfaEncKey,
		federationHTTPClient, slogLogger,
	)
	federationHandler := auth.NewFederationHandler(
		federationSvc, auditLog, "http://localhost:3000", "http://localhost:8080", false,
	)
	federationMgmtHandler := management.NewFederationProviderHandler(
		federationProviderRepo, crypto.Encrypt, mfaEncKey,
	)
	redirectURIRepo := store.NewRedirectURIRepository(db)

	// OIDC ハンドラ
	// issuerBaseURL は serverURL がまだ未確定のため、後で設定する必要がある。
	// httptest.NewServer のアドレスは事前に分からないため、placeholder を使い後で差し替える。
	// → httptest.NewUnstartedServer を使い、URL 確定後に issuerBaseURL を設定する。

	// Echo インスタンスの構築は後で行う（issuerBaseURL が必要なため）
	// 一旦 placeholder URL で組み立てて、httptest.NewUnstartedServer → URL 確定 は不可
	// → Echo の Handler は参照を保持するので、構造体のフィールドを後から変えられない
	// → 方針: httptest.NewUnstartedServer で先にリスナーを確保し、URL を取得してから DI を組み立てる

	// Echo + ルーティング
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// httptest.NewUnstartedServer で先にリスナーを確保
	ts := httptest.NewUnstartedServer(e)
	ts.Start()
	baseURL := ts.URL // 例: "http://127.0.0.1:PORT"

	// issuerBaseURL が確定したので OIDC ハンドラを初期化
	jwksHandler := oidc.NewJWKSHandler(keySvc)
	discoveryHandler := oidc.NewDiscoveryHandler(baseURL, tenantRepo)
	authorizeHandler := oidc.NewAuthorizeHandler(tenantRepo, clientRepo, tenantClientRepo, authCodeRepo, userConsentRepo, authSvc, parRepo, "http://localhost:3000", false)
	parHandler := oidc.NewPARHandler(clientRepo, tenantRepo, tenantClientRepo, parRepo, crypto.VerifyPassword)
	tokenHandler := oidc.NewTokenHandler(
		authCodeRepo, accessTokenRepo, refreshTokenRepo, idTokenRepo,
		clientRepo, tenantRepo, tenantClientRepo, userRepo, tokenSvc,
		crypto.VerifyPassword, crypto.VerifyCodeChallenge,
		jwt.ComputeATHash, jwt.SHA256Hex,
		dpopJTIRepo,
		deviceAuthRepo, sessionRepo,
		tokenSvc, tokenExchangePolicyRepo,
		auditLog, slogLogger,
		baseURL,
		false,
	)
	userInfoHandler := oidc.NewUserInfoHandler(tokenSvc, userRepo, clientRepo, accessTokenRepo, dpopJTIRepo, tokenSvc, baseURL)
	revokeHandler := oidc.NewRevokeHandler(clientRepo, accessTokenRepo, refreshTokenRepo, tokenSvc, crypto.VerifyPassword, jwt.SHA256Hex, auditLog)
	introspectHandler := oidc.NewIntrospectHandler(clientRepo, accessTokenRepo, refreshTokenRepo, tokenSvc, userRepo, crypto.VerifyPassword, jwt.SHA256Hex)

	// Device Authorization
	deviceAuthorizeHandler := oidc.NewDeviceAuthorizeHandler(
		clientRepo, tenantRepo, tenantClientRepo, deviceAuthRepo,
		crypto.VerifyPassword, auditLog, slogLogger, baseURL, "http://localhost:3000",
	)
	deviceVerifyHandler := auth.NewDeviceVerifyHandler(deviceAuthRepo, authSvc, auditLog)

	// Logout
	backChannelClient := &http.Client{Timeout: 10 * time.Second}
	logoutHandler := oidc.NewLogoutHandler(
		tenantRepo, clientRepo, tenantClientRepo,
		clientRepo, postLogoutRedirectURIRepo,
		tokenSvc, tokenSvc,
		sessionRepo, accessTokenRepo, refreshTokenRepo,
		baseURL, "http://localhost:3000",
		backChannelClient, false,
	)
	internalLogoutHandler := auth.NewInternalLogoutHandler(
		sessionRepo, accessTokenRepo, refreshTokenRepo, auditLog, false,
	)

	// OIDC エンドポイント
	e.GET("/jwks", jwksHandler.Handle)
	e.GET("/:tenant_code/.well-known/openid-configuration", discoveryHandler.Handle)
	e.GET("/:tenant_code/authorize", authorizeHandler.Handle)
	e.POST("/:tenant_code/token", tokenHandler.Handle)
	e.GET("/:tenant_code/userinfo", userInfoHandler.Handle)
	e.POST("/:tenant_code/revoke", revokeHandler.Handle)
	e.POST("/:tenant_code/introspect", introspectHandler.Handle)
	e.POST("/:tenant_code/par", parHandler.Handle)
	e.POST("/:tenant_code/device/authorize", deviceAuthorizeHandler.Handle)
	e.GET("/:tenant_code/logout", logoutHandler.Handle)
	e.POST("/:tenant_code/logout", logoutHandler.Handle)

	// Internal API
	e.POST("/internal/login", loginHandler.Handle)
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
	e.GET("/internal/federation/providers", federationHandler.HandleListProviders)
	e.GET("/internal/federation/:provider/initiate", federationHandler.HandleInitiate)
	e.GET("/internal/federation/:provider/callback", federationHandler.HandleCallback)
	e.GET("/internal/device/verify", deviceVerifyHandler.HandleVerify)
	e.POST("/internal/device/approve", deviceVerifyHandler.HandleApprove)
	e.POST("/internal/device/deny", deviceVerifyHandler.HandleDeny)
	e.POST("/internal/email/change-request", emailChangeHandler.HandleRequest)
	e.POST("/internal/email/verify", emailChangeHandler.HandleVerify)
	e.DELETE("/internal/mfa/totp", mfaTOTPDisableHandler.Handle)
	e.POST("/internal/mfa/backup-codes/generate", backupCodeHandler.HandleGenerate)
	e.POST("/internal/mfa/backup-codes/verify", backupCodeHandler.HandleVerify)
	e.GET("/internal/sessions", sessionListHandler.HandleList)
	e.DELETE("/internal/sessions/:id", sessionListHandler.HandleRevoke)

	// Management API
	adminAuthSvc := management.NewAdminAuthService(adminUserRepo, adminSessionRepo, crypto.VerifyPassword)
	adminAuthHandler := management.NewAdminAuthHandler(adminAuthSvc, adminUserRepo, auditLog, slogLogger, false)
	e.POST("/management/v1/auth/login", adminAuthHandler.HandleLogin)
	e.GET("/management/v1/auth/me", adminAuthHandler.HandleMe)
	e.POST("/management/v1/auth/logout", adminAuthHandler.HandleLogout)

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

	mgmtGroup.GET("/tenants/:tenant_id/federation-providers", federationMgmtHandler.HandleList)
	mgmtGroup.POST("/tenants/:tenant_id/federation-providers", federationMgmtHandler.HandleCreate)
	mgmtGroup.GET("/federation-providers/:id", federationMgmtHandler.HandleGet)
	mgmtGroup.PUT("/federation-providers/:id", federationMgmtHandler.HandleUpdate)
	mgmtGroup.DELETE("/federation-providers/:id", federationMgmtHandler.HandleDelete)

	incidentHandler := management.NewIncidentHandler(sessionRepo, accessTokenRepo, refreshTokenRepo, userRepo)
	mgmtGroup.POST("/incidents/revoke-all-tokens", incidentHandler.HandleRevokeAll)
	mgmtGroup.POST("/incidents/revoke-tenant-tokens", incidentHandler.HandleRevokeTenant)
	mgmtGroup.POST("/incidents/revoke-user-tokens", incidentHandler.HandleRevokeUser)
	mgmtGroup.POST("/users/:user_id/unlock", incidentHandler.HandleUnlockUser)

	return ts
}
