package jwt

import (
	"context"
	"crypto"

	"github.com/lestrrat-go/jwx/v3/jwk"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

type SignKeyRepository interface {
	Create(ctx context.Context, key *model.SignKey) error
	FindActive(ctx context.Context) (*model.SignKey, error)
	FindAllActive(ctx context.Context) ([]model.SignKey, error)
}

type KeyProvider interface {
	GetActiveSigningKey(ctx context.Context) (kid string, privateKey crypto.PrivateKey, err error)
	GetJWKSet(ctx context.Context) (jwk.Set, error)
}
