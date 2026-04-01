package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

// InitialAccessTokenRepository は initial_access_tokens テーブルの GORM 実装。
type InitialAccessTokenRepository struct {
	db *gorm.DB
}

// NewInitialAccessTokenRepository は InitialAccessTokenRepository を生成する。
func NewInitialAccessTokenRepository(db *gorm.DB) *InitialAccessTokenRepository {
	return &InitialAccessTokenRepository{db: db}
}

// FindByTokenHash はトークンハッシュで IAT を検索する。見つからない場合は (nil, nil)。
func (r *InitialAccessTokenRepository) FindByTokenHash(ctx context.Context, hash string) (*model.InitialAccessToken, error) {
	var iat model.InitialAccessToken
	result := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&iat)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &iat, nil
}

// FindByID は UUID で IAT を検索する。見つからない場合は (nil, nil)。
func (r *InitialAccessTokenRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.InitialAccessToken, error) {
	var iat model.InitialAccessToken
	result := r.db.WithContext(ctx).First(&iat, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &iat, nil
}

// ListByTenantID はテナントに属する IAT をページネーション付きで返す。
func (r *InitialAccessTokenRepository) ListByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]model.InitialAccessToken, int64, error) {
	var tokens []model.InitialAccessToken
	var count int64

	baseQuery := r.db.WithContext(ctx).Model(&model.InitialAccessToken{}).Where("tenant_id = ?", tenantID)
	if err := baseQuery.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := baseQuery.
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&tokens).Error; err != nil {
		return nil, 0, err
	}
	return tokens, count, nil
}

// Create は新しい IAT を永続化する。
func (r *InitialAccessTokenRepository) Create(ctx context.Context, iat *model.InitialAccessToken) error {
	return r.db.WithContext(ctx).Create(iat).Error
}

// IncrementUsedCount は used_count を 1 加算する。
func (r *InitialAccessTokenRepository) IncrementUsedCount(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.InitialAccessToken{}).
		Where("id = ?", id).
		UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error
}

// Revoke は revoked_at を設定して IAT を無効化する。
func (r *InitialAccessTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.InitialAccessToken{}).
		Where("id = ?", id).
		Update("revoked_at", now).Error
}
