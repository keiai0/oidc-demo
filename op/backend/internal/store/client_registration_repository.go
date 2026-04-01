package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

// ClientRegistrationRepository は client_registrations テーブルの GORM 実装。
type ClientRegistrationRepository struct {
	db *gorm.DB
}

// NewClientRegistrationRepository は ClientRegistrationRepository を生成する。
func NewClientRegistrationRepository(db *gorm.DB) *ClientRegistrationRepository {
	return &ClientRegistrationRepository{db: db}
}

// FindByClientID はクライアント DB ID で登録情報を検索する。見つからない場合は (nil, nil)。
func (r *ClientRegistrationRepository) FindByClientID(ctx context.Context, clientID uuid.UUID) (*model.ClientRegistration, error) {
	var reg model.ClientRegistration
	result := r.db.WithContext(ctx).Where("client_id = ?", clientID).First(&reg)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &reg, nil
}

// FindByRegistrationTokenHash は Registration Access Token のハッシュで検索する。
func (r *ClientRegistrationRepository) FindByRegistrationTokenHash(ctx context.Context, hash string) (*model.ClientRegistration, error) {
	var reg model.ClientRegistration
	result := r.db.WithContext(ctx).Where("registration_access_token_hash = ?", hash).First(&reg)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &reg, nil
}

// Create は新しい登録情報を永続化する。
func (r *ClientRegistrationRepository) Create(ctx context.Context, reg *model.ClientRegistration) error {
	return r.db.WithContext(ctx).Create(reg).Error
}

// Update は登録情報を更新する。
func (r *ClientRegistrationRepository) Update(ctx context.Context, reg *model.ClientRegistration) error {
	return r.db.WithContext(ctx).Save(reg).Error
}

// Delete は登録情報を削除する。
func (r *ClientRegistrationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.ClientRegistration{}, "id = ?", id).Error
}
