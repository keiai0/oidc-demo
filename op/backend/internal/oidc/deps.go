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

type (
	VerifyPasswordFunc      func(password, hash string) (bool, error)
	VerifyCodeChallengeFunc func(verifier, challenge string) bool
	ComputeATHashFunc       func(accessToken string) string
	SHA256HexFunc           func(s string) string
)
