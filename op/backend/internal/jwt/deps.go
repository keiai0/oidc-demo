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
	// FindAllByStatus は指定ステータスの全鍵を返す。
	FindAllByStatus(ctx context.Context, status string) ([]model.SignKey, error)
	// UpdateStatus は指定 kid の鍵ステータスと rotated_at を更新する。
	UpdateStatus(ctx context.Context, kid string, status string) error
	// UpdateStatusBulk は指定ステータスの全鍵を newStatus に一括更新する。
	UpdateStatusBulk(ctx context.Context, fromStatus string, toStatus string) error
	// DeleteByStatus は指定ステータスの全鍵を削除する。
	DeleteByStatus(ctx context.Context, status string) error
}

type KeyProvider interface {
	GetActiveSigningKey(ctx context.Context) (kid string, privateKey crypto.PrivateKey, err error)
	GetJWKSet(ctx context.Context) (jwk.Set, error)
}
