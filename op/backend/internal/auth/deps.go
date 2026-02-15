package auth

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

type TenantFinder interface {
	FindByCode(ctx context.Context, code string) (*model.Tenant, error)
}

type UserFinder interface {
	FindByTenantAndLoginID(ctx context.Context, tenantID uuid.UUID, loginID string) (*model.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	UpdateLastLoginAt(ctx context.Context, id uuid.UUID, t time.Time) error
}

type SessionStore interface {
	Create(ctx context.Context, session *model.Session) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Session, error)
}

type PasswordVerifyFunc func(password, hash string) (bool, error)
