package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

// ExternalIdPCredentialRepository は外部 IdP クレデンシャルの永続化を担当する。
type ExternalIdPCredentialRepository struct {
	db *gorm.DB
}

func NewExternalIdPCredentialRepository(db *gorm.DB) *ExternalIdPCredentialRepository {
	return &ExternalIdPCredentialRepository{db: db}
}

func (r *ExternalIdPCredentialRepository) Create(ctx context.Context, cred *model.ExternalIdPCredential) error {
	return r.db.WithContext(ctx).Create(cred).Error
}

// FindByProviderAndSubject は外部 IdP の provider_id と sub で検索する。
// Credential → User のリレーションもプリロードする。
func (r *ExternalIdPCredentialRepository) FindByProviderAndSubject(ctx context.Context, providerID uuid.UUID, subject string) (*model.ExternalIdPCredential, error) {
	var cred model.ExternalIdPCredential
	result := r.db.WithContext(ctx).
		Preload("Credential").
		Where("provider_id = ? AND provider_subject = ?", providerID, subject).
		First(&cred)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &cred, nil
}
