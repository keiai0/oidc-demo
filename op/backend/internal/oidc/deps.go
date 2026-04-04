package oidc

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwk"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

type TenantFinder interface {
	FindByCode(ctx context.Context, code string) (*model.Tenant, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error)
}

type ClientFinder interface {
	FindByClientID(ctx context.Context, clientID string) (*model.Client, error)
	FindByClientIDWithRedirectURIs(ctx context.Context, clientID string) (*model.Client, error)
}

// TenantClientChecker はテナントとクライアントの紐づきを検証する。
type TenantClientChecker interface {
	// ExistsByTenantAndClient はテナントでクライアントが有効かどうかを返す。
	ExistsByTenantAndClient(ctx context.Context, tenantID, clientID uuid.UUID) (bool, error)
}

type AuthorizationCodeStore interface {
	Create(ctx context.Context, code *model.AuthorizationCode) error
	FindByCode(ctx context.Context, code string) (*model.AuthorizationCode, error)
	MarkAsUsed(ctx context.Context, id uuid.UUID) error
}

type AccessTokenStore interface {
	Create(ctx context.Context, token *model.AccessToken) error
	FindByJTI(ctx context.Context, jti string) (*model.AccessToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeBySessionID(ctx context.Context, sessionID uuid.UUID) error
}

type RefreshTokenStore interface {
	Create(ctx context.Context, token *model.RefreshToken) error
	FindByTokenHash(ctx context.Context, hash string) (*model.RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	RevokeBySessionID(ctx context.Context, sessionID uuid.UUID) error
	MarkReuseDetected(ctx context.Context, id uuid.UUID) error
}

type IDTokenCreator interface {
	Create(ctx context.Context, token *model.IDToken) error
}

type UserFinder interface {
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type SessionValidator interface {
	ValidateSession(ctx context.Context, sessionID uuid.UUID) (*model.Session, error)
}

// ConsentStore は同意記録の永続化操作を定義する。
type ConsentStore interface {
	// FindByUserAndClient はユーザーとクライアントの組み合わせで有効な同意記録を検索する。
	FindByUserAndClient(ctx context.Context, userID, clientID uuid.UUID) (*model.UserConsent, error)
}

type KeySetProvider interface {
	GetJWKSet(ctx context.Context) (jwk.Set, error)
}

type TokenSigner interface {
	SignIDToken(ctx context.Context, claims *model.IDTokenClaims, lifetime time.Duration) (jti string, signedToken string, err error)
	SignAccessToken(ctx context.Context, claims *model.AccessTokenClaims, lifetime time.Duration) (jti string, signedToken string, err error)
	GenerateRefreshToken() (token string, tokenHash string, err error)
}

type TokenValidator interface {
	ValidateAccessToken(ctx context.Context, tokenString string) (*model.AccessTokenResult, error)
}

// IDTokenHintValidator は id_token_hint の署名検証と claims 抽出を行う。
type IDTokenHintValidator interface {
	// ValidateIDTokenHint は署名を検証し、exp はスキップする（期限切れの ID トークンが渡される場合がある）。
	ValidateIDTokenHint(ctx context.Context, tokenString string) (*model.IDTokenHintResult, error)
}

// LogoutTokenSigner は Back-Channel Logout 用の logout_token を署名する。
type LogoutTokenSigner interface {
	SignLogoutToken(ctx context.Context, claims *model.LogoutTokenClaims) (string, error)
}

// SessionRevoker はセッションの検索と失効操作を定義する。
type SessionRevoker interface {
	FindByID(ctx context.Context, id uuid.UUID) (*model.Session, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

// ClientsByTenantLister はテナントに属する SLO 対象クライアントの一覧取得を定義する。
type ClientsByTenantLister interface {
	ListByTenantIDWithLogoutURIs(ctx context.Context, tenantID uuid.UUID) ([]model.Client, error)
}

// PostLogoutRedirectURILister はクライアントの post_logout_redirect_uri 一覧取得を定義する。
type PostLogoutRedirectURILister interface {
	ListByClientID(ctx context.Context, clientDBID uuid.UUID) ([]model.PostLogoutRedirectURI, error)
}

// DPoPJTIStore は DPoP proof JWT の JTI リプレイ防止キャッシュを操作する (RFC 9449)。
type DPoPJTIStore interface {
	// Exists は JTI がキャッシュに存在するかを返す。
	Exists(ctx context.Context, jti string) (bool, error)
	// Create は JTI をキャッシュに登録する。
	Create(ctx context.Context, jti string) error
}

// PARStore は Pushed Authorization Request (RFC 9126) の永続化操作を定義する。
type PARStore interface {
	Create(ctx context.Context, par *model.PushedAuthorizationRequest) error
	FindByRequestURI(ctx context.Context, requestURI string) (*model.PushedAuthorizationRequest, error)
	MarkAsUsed(ctx context.Context, id uuid.UUID) error
}

// UserinfoJWTSigner は userinfo レスポンスを署名付き JWT として返す (OIDC Core Section 5.3.2)。
type UserinfoJWTSigner interface {
	SignUserInfoResponse(ctx context.Context, claims map[string]interface{}) (string, error)
}

// DeviceAuthorizationRequestStore は Device Authorization Grant (RFC 8628) のリクエスト永続化を定義する。
type DeviceAuthorizationRequestStore interface {
	Create(ctx context.Context, req *model.DeviceAuthorizationRequest) error
	FindByDeviceCode(ctx context.Context, deviceCode string) (*model.DeviceAuthorizationRequest, error)
	FindByUserCode(ctx context.Context, userCode string) (*model.DeviceAuthorizationRequest, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, sessionID *uuid.UUID) error
	UpdateLastPolledAt(ctx context.Context, id uuid.UUID, t time.Time) error
	IncrementPollInterval(ctx context.Context, id uuid.UUID, incrementSec int) error
}

// SessionFinder はセッションの検索操作を定義する。
type SessionFinder interface {
	FindByID(ctx context.Context, id uuid.UUID) (*model.Session, error)
}

// TokenExchangePolicyFinder は Token Exchange ポリシーの検索を定義する (RFC 8693)。
type TokenExchangePolicyFinder interface {
	// FindByClientID はクライアント DB ID でポリシーを検索する。見つからなければ (nil, nil)。
	FindByClientID(ctx context.Context, clientID uuid.UUID) (*model.TokenExchangePolicy, error)
}

// ClientCreator はクライアントの新規作成を定義する (Dynamic Client Registration)。
type ClientCreator interface {
	Create(ctx context.Context, client *model.Client) error
}

// ClientUpdater はクライアントの更新を定義する (Dynamic Client Registration)。
type ClientUpdater interface {
	Update(ctx context.Context, client *model.Client) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

// InitialAccessTokenFinder は Initial Access Token の検索と使用回数更新を定義する (RFC 7591)。
type InitialAccessTokenFinder interface {
	// FindByTokenHash はトークンハッシュで IAT を検索する。見つからなければ (nil, nil)。
	FindByTokenHash(ctx context.Context, hash string) (*model.InitialAccessToken, error)
	// IncrementUsedCount は使用回数を 1 加算する。
	IncrementUsedCount(ctx context.Context, id uuid.UUID) error
}

// ClientRegistrationStore は Dynamic Client Registration のメタデータ永続化を定義する (RFC 7591/7592)。
type ClientRegistrationStore interface {
	Create(ctx context.Context, reg *model.ClientRegistration) error
	FindByClientID(ctx context.Context, clientID uuid.UUID) (*model.ClientRegistration, error)
	FindByRegistrationTokenHash(ctx context.Context, hash string) (*model.ClientRegistration, error)
	Update(ctx context.Context, reg *model.ClientRegistration) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// TenantClientCreator はテナント-クライアント紐づけの作成を定義する。
type TenantClientCreator interface {
	Create(ctx context.Context, tc *model.TenantClient) error
}

// RedirectURICreator はリダイレクト URI の永続化を定義する。
type RedirectURICreator interface {
	Create(ctx context.Context, uri *model.RedirectURI) error
	DeleteByClientID(ctx context.Context, clientDBID uuid.UUID) error
}

// AuthorizationDetailTypeFinder は認可詳細タイプの検索を定義する (RFC 9396)。
type AuthorizationDetailTypeFinder interface {
	// FindByTenantIDAndType はテナント ID とタイプ名で認可詳細タイプを検索する。見つからなければ (nil, nil)。
	FindByTenantIDAndType(ctx context.Context, tenantID uuid.UUID, typeName string) (*model.AuthorizationDetailType, error)
	// ListByTenantID はテナントに属する認可詳細タイプの一覧を返す。
	ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]model.AuthorizationDetailType, error)
}

type (
	VerifyPasswordFunc      func(password, hash string) (bool, error)
	VerifyCodeChallengeFunc func(verifier, challenge string) bool
	ComputeATHashFunc       func(accessToken string) string
	SHA256HexFunc           func(s string) string
	HashPasswordFunc        func(password string) (string, error)
)
