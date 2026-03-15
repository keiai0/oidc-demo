package store

import (
	"context"
	"errors"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

// DPoPJTICacheRepository は DPoP proof JWT の JTI リプレイ防止キャッシュを管理する。
type DPoPJTICacheRepository struct {
	db *gorm.DB
}

func NewDPoPJTICacheRepository(db *gorm.DB) *DPoPJTICacheRepository {
	return &DPoPJTICacheRepository{db: db}
}

// Exists は指定した JTI がキャッシュに存在するかを返す。
func (r *DPoPJTICacheRepository) Exists(ctx context.Context, jti string) (bool, error) {
	var cache model.DPoPJTICache
	result := r.db.WithContext(ctx).Where("jti = ?", jti).First(&cache)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, result.Error
	}
	return true, nil
}

// Create は JTI をキャッシュに登録する。
func (r *DPoPJTICacheRepository) Create(ctx context.Context, jti string) error {
	cache := &model.DPoPJTICache{JTI: jti}
	return r.db.WithContext(ctx).Create(cache).Error
}
