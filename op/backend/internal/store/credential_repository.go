package store

import (
	"context"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

// CredentialRepository はクレデンシャル（認証手段）の永続化を担当する。
type CredentialRepository struct {
	db *gorm.DB
}

func NewCredentialRepository(db *gorm.DB) *CredentialRepository {
	return &CredentialRepository{db: db}
}

func (r *CredentialRepository) Create(ctx context.Context, cred *model.Credential) error {
	return r.db.WithContext(ctx).Create(cred).Error
}
