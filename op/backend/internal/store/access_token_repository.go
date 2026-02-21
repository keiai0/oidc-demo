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

// RevokeAll は全ての有効なアクセストークンを失効させる。影響行数を返す。
func (r *AccessTokenRepository) RevokeAll(ctx context.Context) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.AccessToken{}).
		Where("revoked_at IS NULL").
		Update("revoked_at", now)
	return result.RowsAffected, result.Error
}

// RevokeByTenantID はテナントに属する全ての有効なアクセストークンを失効させる（セッション経由）。
func (r *AccessTokenRepository) RevokeByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.AccessToken{}).
		Where("revoked_at IS NULL AND session_id IN (?)",
			r.db.Model(&model.Session{}).Select("id").Where("tenant_id = ?", tenantID),
		).
		Update("revoked_at", now)
	return result.RowsAffected, result.Error
}

// RevokeByUserID はユーザーに属する全ての有効なアクセストークンを失効させる（セッション経由）。
func (r *AccessTokenRepository) RevokeByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.AccessToken{}).
		Where("revoked_at IS NULL AND session_id IN (?)",
			r.db.Model(&model.Session{}).Select("id").Where("user_id = ?", userID),
		).
		Update("revoked_at", now)
	return result.RowsAffected, result.Error
}
