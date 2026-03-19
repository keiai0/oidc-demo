package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// FederationProviderFinder は federation_providers テーブルの検索操作を定義する。
type FederationProviderFinder interface {
	FindByTenantAndName(ctx context.Context, tenantID uuid.UUID, name string) (*model.FederationProvider, error)
	ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]model.FederationProvider, error)
}

// ExternalIdPCredentialStore は外部 IdP クレデンシャルの永続化操作を定義する。
type ExternalIdPCredentialStore interface {
	FindByProviderAndSubject(ctx context.Context, providerID uuid.UUID, subject string) (*model.ExternalIdPCredential, error)
	Create(ctx context.Context, cred *model.ExternalIdPCredential) error
}

// CredentialCreator はクレデンシャルの作成操作を定義する。
type CredentialCreator interface {
	Create(ctx context.Context, cred *model.Credential) error
}

// UserCreator はユーザーの作成操作を定義する。
type UserCreator interface {
	Create(ctx context.Context, user *model.User) error
}

// oidcDiscoveryDoc は外部 IdP の OIDC Discovery レスポンスの一部。
type oidcDiscoveryDoc struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
	Issuer                string `json:"issuer"`
}

// externalTokenResponse は外部 IdP のトークンレスポンス。
type externalTokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
}

// FederationService は外部 IdP 連携の認証フローを処理する。
type FederationService struct {
	providerFinder FederationProviderFinder
	extCredStore   ExternalIdPCredentialStore
	credCreator    CredentialCreator
	userCreator    UserCreator
	sessionStore   SessionStore
	tenantFinder   TenantFinder
	decrypt        DecryptFunc
	encKey         []byte
	httpClient     *http.Client
	logger         *slog.Logger
}

// NewFederationService は FederationService を生成する。
func NewFederationService(
	providerFinder FederationProviderFinder,
	extCredStore ExternalIdPCredentialStore,
	credCreator CredentialCreator,
	userCreator UserCreator,
	sessionStore SessionStore,
	tenantFinder TenantFinder,
	decrypt DecryptFunc,
	encKey []byte,
	httpClient *http.Client,
	logger *slog.Logger,
) *FederationService {
	return &FederationService{
		providerFinder: providerFinder,
		extCredStore:   extCredStore,
		credCreator:    credCreator,
		userCreator:    userCreator,
		sessionStore:   sessionStore,
		tenantFinder:   tenantFinder,
		decrypt:        decrypt,
		encKey:         encKey,
		httpClient:     httpClient,
		logger:         logger,
	}
}

// FederationInitResult は認証開始時の結果。
type FederationInitResult struct {
	AuthorizationURL string
}

// InitiateFederation は外部 IdP への認可リクエスト URL を構築する。
func (s *FederationService) InitiateFederation(
	ctx context.Context,
	tenantCode, providerName, state, nonce, redirectURI string,
) (*FederationInitResult, error) {
	tenant, err := s.tenantFinder.FindByCode(ctx, tenantCode)
	if err != nil || tenant == nil {
		return nil, fmt.Errorf("tenant not found: %s", tenantCode)
	}

	provider, err := s.providerFinder.FindByTenantAndName(ctx, tenant.ID, providerName)
	if err != nil || provider == nil || !provider.IsActive() {
		return nil, fmt.Errorf("federation provider not found: %s", providerName)
	}

	// 外部 IdP の Discovery ドキュメントを取得
	discovery, err := s.fetchDiscovery(ctx, provider.Issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch discovery: %w", err)
	}

	// 認可 URL 構築
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", provider.ClientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", provider.Scopes)
	params.Set("state", state)
	params.Set("nonce", nonce)

	authURL := discovery.AuthorizationEndpoint + "?" + params.Encode()

	return &FederationInitResult{AuthorizationURL: authURL}, nil
}

// FederationCallbackResult はコールバック処理の結果。
type FederationCallbackResult struct {
	Session *model.Session
	IsNew   bool // JIT プロビジョニングで新規作成されたか
}

// HandleCallback は外部 IdP からのコールバックを処理する。
// 認可コードをトークンに交換し、ID トークンを検証し、ユーザーを紐付け/作成する。
func (s *FederationService) HandleCallback(
	ctx context.Context,
	tenantCode, providerName, code, redirectURI string,
	ipAddress, userAgent string,
) (*FederationCallbackResult, error) {
	tenant, err := s.tenantFinder.FindByCode(ctx, tenantCode)
	if err != nil || tenant == nil {
		return nil, fmt.Errorf("tenant not found: %s", tenantCode)
	}

	provider, err := s.providerFinder.FindByTenantAndName(ctx, tenant.ID, providerName)
	if err != nil || provider == nil || !provider.IsActive() {
		return nil, fmt.Errorf("federation provider not found: %s", providerName)
	}

	// Discovery ドキュメント取得
	discovery, err := s.fetchDiscovery(ctx, provider.Issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch discovery: %w", err)
	}

	// client_secret 復号（AES-256-GCM）
	clientSecretBytes, err := s.decrypt(provider.ClientSecretEnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt client secret: %w", err)
	}

	// 認可コードをトークンに交換
	tokenResp, err := s.exchangeCode(ctx, discovery.TokenEndpoint, code, redirectURI, provider.ClientID, string(clientSecretBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// ID トークン検証
	sub, email, name, err := s.validateExternalIDToken(ctx, tokenResp.IDToken, discovery, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to validate external id token: %w", err)
	}

	// 既存の外部クレデンシャル検索
	extCred, err := s.extCredStore.FindByProviderAndSubject(ctx, provider.ID, sub)
	if err != nil {
		return nil, fmt.Errorf("failed to find external credential: %w", err)
	}

	var userID uuid.UUID
	isNew := false

	if extCred != nil {
		// 既存ユーザー
		userID = extCred.Credential.UserID
	} else {
		// 新規: JIT プロビジョニング
		if !provider.AutoProvision {
			return nil, fmt.Errorf("user not found and auto_provision is disabled")
		}

		user, err := s.provisionUser(ctx, tenant.ID, provider.ID, sub, email, name)
		if err != nil {
			return nil, fmt.Errorf("failed to provision user: %w", err)
		}
		userID = user.ID
		isNew = true
	}

	// セッション作成
	session := &model.Session{
		UserID:    userID,
		TenantID:  tenant.ID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		AuthTime:  time.Now(),
		AMR:       model.StringSlice{"fed"},
		ACR:       "urn:mace:incommon:iap:bronze",
		ExpiresAt: time.Now().Add(time.Duration(tenant.SessionLifetime) * time.Second),
	}
	if err := s.sessionStore.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &FederationCallbackResult{Session: session, IsNew: isNew}, nil
}

// ListProviders はテナントのアクティブな連携プロバイダ一覧を返す。
func (s *FederationService) ListProviders(ctx context.Context, tenantCode string) ([]model.FederationProvider, error) {
	tenant, err := s.tenantFinder.FindByCode(ctx, tenantCode)
	if err != nil || tenant == nil {
		return nil, fmt.Errorf("tenant not found: %s", tenantCode)
	}
	return s.providerFinder.ListByTenantID(ctx, tenant.ID)
}

// provisionUser は JIT プロビジョニングでユーザーを作成する。
func (s *FederationService) provisionUser(
	ctx context.Context,
	tenantID, providerID uuid.UUID,
	sub, email, name string,
) (*model.User, error) {
	// login_id は provider名+sub で一意にする
	loginID := "fed:" + providerID.String()[:8] + ":" + sub

	user := &model.User{
		TenantID:      tenantID,
		LoginID:       loginID,
		Email:         email,
		EmailVerified: true, // 外部 IdP が検証済みと仮定
		Name:          &name,
		Status:        "active",
	}
	if err := s.userCreator.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Credential (type=oidc_provider) 作成
	cred := &model.Credential{
		UserID: user.ID,
		Type:   "oidc_provider",
	}
	if err := s.credCreator.Create(ctx, cred); err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	// ExternalIdPCredential 作成
	extCred := &model.ExternalIdPCredential{
		CredentialID:    cred.ID,
		ProviderID:      providerID,
		ProviderSubject: sub,
	}
	if err := s.extCredStore.Create(ctx, extCred); err != nil {
		return nil, fmt.Errorf("failed to create external idp credential: %w", err)
	}

	return user, nil
}

// fetchDiscovery は外部 IdP の OIDC Discovery ドキュメントを取得する。
func (s *FederationService) fetchDiscovery(ctx context.Context, issuer string) (*oidcDiscoveryDoc, error) {
	discoveryURL := strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch discovery: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery returned status %d", resp.StatusCode)
	}

	var doc oidcDiscoveryDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("failed to decode discovery: %w", err)
	}
	return &doc, nil
}

// exchangeCode は認可コードを外部 IdP のトークンに交換する。
func (s *FederationService) exchangeCode(
	ctx context.Context,
	tokenEndpoint, code, redirectURI, clientID, clientSecret string,
) (*externalTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp externalTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}
	return &tokenResp, nil
}

// validateExternalIDToken は外部 IdP の ID トークンの署名と claims を検証する。
// 返り値: sub, email, name
func (s *FederationService) validateExternalIDToken(
	ctx context.Context,
	idTokenStr string,
	discovery *oidcDiscoveryDoc,
	provider *model.FederationProvider,
) (sub, email, name string, err error) {
	// 外部 IdP の JWKS 取得
	jwksReq, err := http.NewRequestWithContext(ctx, http.MethodGet, discovery.JWKSURI, nil)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create jwks request: %w", err)
	}
	jwksResp, err := s.httpClient.Do(jwksReq)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to fetch jwks: %w", err)
	}
	defer jwksResp.Body.Close()

	jwksBody, err := io.ReadAll(jwksResp.Body)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to read jwks: %w", err)
	}

	keySet, err := jwk.ParseKey(jwksBody)
	if err != nil {
		// JWKS Set として再パース
		set, err2 := jwk.Parse(jwksBody)
		if err2 != nil {
			return "", "", "", fmt.Errorf("failed to parse jwks: %w (key error: %v)", err2, err)
		}
		// Set を使って署名検証
		token, err2 := jwt.Parse([]byte(idTokenStr),
			jwt.WithKeySet(set, jws.WithInferAlgorithmFromKey(true)),
		)
		if err2 != nil {
			return "", "", "", fmt.Errorf("failed to verify id token signature: %w", err2)
		}
		return extractClaims(token, discovery.Issuer, provider.ClientID)
	}

	// 単一キーで署名検証
	alg, _ := keySet.Algorithm()
	token, err := jwt.Parse([]byte(idTokenStr),
		jwt.WithKey(alg, keySet),
	)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to verify id token: %w", err)
	}
	return extractClaims(token, discovery.Issuer, provider.ClientID)
}

// extractClaims は JWT トークンから sub, email, name を抽出し、iss と aud を検証する。
// lestrrat-go/jwx v3 API: getter メソッドは (value, bool) を返す。
func extractClaims(token jwt.Token, expectedIssuer, expectedAud string) (sub, email, name string, err error) {
	// issuer 検証
	iss, ok := token.Issuer()
	if !ok || iss != expectedIssuer {
		return "", "", "", fmt.Errorf("issuer mismatch: expected %s, got %s", expectedIssuer, iss)
	}

	// audience 検証
	audiences, ok := token.Audience()
	if !ok {
		return "", "", "", fmt.Errorf("audience claim is missing")
	}
	found := false
	for _, aud := range audiences {
		if aud == expectedAud {
			found = true
			break
		}
	}
	if !found {
		return "", "", "", fmt.Errorf("audience mismatch: expected %s", expectedAud)
	}

	// subject
	sub, ok = token.Subject()
	if !ok || sub == "" {
		return "", "", "", fmt.Errorf("sub claim is empty")
	}

	// email と name は optional claims
	var emailVal string
	if err := token.Get("email", &emailVal); err == nil {
		email = emailVal
	}
	var nameVal string
	if err := token.Get("name", &nameVal); err == nil {
		name = nameVal
	}

	return sub, email, name, nil
}
