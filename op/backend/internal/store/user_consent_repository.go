package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

type UserConsentRepository struct {
	db *gorm.DB
}

func NewUserConsentRepository(db *gorm.DB) *UserConsentRepository {
	return &UserConsentRepository{db: db}
}

// FindByUserAndClient はユーザーとクライアントの組み合わせで同意記録を検索する。
func (r *UserConsentRepository) FindByUserAndClient(ctx context.Context, userID, clientID uuid.UUID) (*model.UserConsent, error) {
	var consent model.UserConsent
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND client_id = ? AND revoked_at IS NULL", userID, clientID).
		First(&consent)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &consent, nil
}

// Upsert は同意を作成または更新する。
func (r *UserConsentRepository) Upsert(ctx context.Context, consent *model.UserConsent) error {
	var existing model.UserConsent
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND client_id = ?", consent.UserID, consent.ClientID).
		First(&existing)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return r.db.WithContext(ctx).Create(consent).Error
		}
		return result.Error
	}

	// 既存レコードを更新
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&existing).
		Updates(map[string]interface{}{
			"scopes":     consent.Scopes,
			"granted_at": now,
			"revoked_at": nil,
			"updated_at": now,
		}).Error
}

// Revoke は同意を失効させる。
func (r *UserConsentRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.UserConsent{}).
		Where("id = ?", id).
		Update("revoked_at", now).Error
}
