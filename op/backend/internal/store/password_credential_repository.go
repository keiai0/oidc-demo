package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

type PasswordCredentialRepository struct {
	db *gorm.DB
}

func NewPasswordCredentialRepository(db *gorm.DB) *PasswordCredentialRepository {
	return &PasswordCredentialRepository{db: db}
}

// UpdateHash はパスワードクレデンシャルのハッシュを更新する。
func (r *PasswordCredentialRepository) UpdateHash(ctx context.Context, credentialID uuid.UUID, newHash string) error {
	return r.db.WithContext(ctx).
		Model(&model.PasswordCredential{}).
		Where("credential_id = ?", credentialID).
		Update("password_hash", newHash).Error
}
