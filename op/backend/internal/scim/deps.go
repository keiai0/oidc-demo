package scim

import (
	"context"

	"github.com/google/uuid"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// UserStore は SCIM ユーザー操作で使用するリポジトリインターフェース。
type UserStore interface {
	Create(ctx context.Context, user *model.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	FindByTenantAndLoginID(ctx context.Context, tenantID uuid.UUID, loginID string) (*model.User, error)
	FindByTenantAndExternalID(ctx context.Context, tenantID uuid.UUID, externalID string) (*model.User, error)
	ListByTenantIDWithFilter(ctx context.Context, tenantID uuid.UUID, filterAttr, filterValue string, offset, limit int) ([]model.User, int64, error)
	UpdateSCIM(ctx context.Context, user *model.User) error
	Deactivate(ctx context.Context, id uuid.UUID) error
}

// TenantFinder はテナント検索インターフェース。
type TenantFinder interface {
	FindByCode(ctx context.Context, code string) (*model.Tenant, error)
}

// TokenValidator はアクセストークン検証インターフェース。
type TokenValidator interface {
	ValidateAccessToken(ctx context.Context, tokenString string) (*model.AccessTokenResult, error)
}

type HashPasswordFunc func(password string) (string, error)
