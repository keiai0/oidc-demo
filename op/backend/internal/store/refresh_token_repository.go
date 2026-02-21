package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

type RefreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token *model.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *RefreshTokenRepository) FindByTokenHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	var token model.RefreshToken
	result := r.db.WithContext(ctx).
		Preload("Session").
		Preload("AccessToken").
		Preload("AccessToken.Client").
		Where("token_hash = ?", hash).
		First(&token)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &token, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now).Error
}

func (r *RefreshTokenRepository) RevokeBySessionID(ctx context.Context, sessionID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("session_id = ? AND revoked_at IS NULL", sessionID).
		Update("revoked_at", now).Error
}

func (r *RefreshTokenRepository) MarkReuseDetected(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("id = ?", id).
		Update("reuse_detected_at", now).Error
}

// RevokeAll は全ての有効なリフレッシュトークンを失効させる。影響行数を返す。
func (r *RefreshTokenRepository) RevokeAll(ctx context.Context) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("revoked_at IS NULL").
		Update("revoked_at", now)
	return result.RowsAffected, result.Error
}

// RevokeByTenantID はテナントに属する全ての有効なリフレッシュトークンを失効させる（セッション経由）。
func (r *RefreshTokenRepository) RevokeByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("revoked_at IS NULL AND session_id IN (?)",
			r.db.Model(&model.Session{}).Select("id").Where("tenant_id = ?", tenantID),
		).
		Update("revoked_at", now)
	return result.RowsAffected, result.Error
}

// RevokeByUserID はユーザーに属する全ての有効なリフレッシュトークンを失効させる（セッション経由）。
func (r *RefreshTokenRepository) RevokeByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("revoked_at IS NULL AND session_id IN (?)",
			r.db.Model(&model.Session{}).Select("id").Where("user_id = ?", userID),
		).
		Update("revoked_at", now)
	return result.RowsAffected, result.Error
}
