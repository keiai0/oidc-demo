package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

type AccessTokenRepository struct {
	db *gorm.DB
}

func NewAccessTokenRepository(db *gorm.DB) *AccessTokenRepository {
	return &AccessTokenRepository{db: db}
}

func (r *AccessTokenRepository) Create(ctx context.Context, token *model.AccessToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *AccessTokenRepository) FindByJTI(ctx context.Context, jti string) (*model.AccessToken, error) {
	var token model.AccessToken
	result := r.db.WithContext(ctx).
		Preload("Session").
		Preload("Client").
		Where("jti = ?", jti).
		First(&token)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &token, nil
}

func (r *AccessTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.AccessToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now).Error
}

func (r *AccessTokenRepository) RevokeBySessionID(ctx context.Context, sessionID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.AccessToken{}).
		Where("session_id = ? AND revoked_at IS NULL", sessionID).
		Update("revoked_at", now).Error
}
