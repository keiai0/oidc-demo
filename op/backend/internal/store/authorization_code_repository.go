package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

type AuthorizationCodeRepository struct {
	db *gorm.DB
}

func NewAuthorizationCodeRepository(db *gorm.DB) *AuthorizationCodeRepository {
	return &AuthorizationCodeRepository{db: db}
}

func (r *AuthorizationCodeRepository) Create(ctx context.Context, code *model.AuthorizationCode) error {
	return r.db.WithContext(ctx).Create(code).Error
}

func (r *AuthorizationCodeRepository) FindByCode(ctx context.Context, code string) (*model.AuthorizationCode, error) {
	var ac model.AuthorizationCode
	result := r.db.WithContext(ctx).
		Preload("Session").
		Preload("Client").
		Where("code = ?", code).
		First(&ac)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &ac, nil
}

func (r *AuthorizationCodeRepository) MarkAsUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.AuthorizationCode{}).
		Where("id = ?", id).
		Update("used_at", now).Error
}
