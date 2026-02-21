package store

import (
	"context"
	"errors"
	"time"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

type SignKeyRepository struct {
	db *gorm.DB
}

func NewSignKeyRepository(db *gorm.DB) *SignKeyRepository {
	return &SignKeyRepository{db: db}
}

func (r *SignKeyRepository) Create(ctx context.Context, key *model.SignKey) error {
	return r.db.WithContext(ctx).Create(key).Error
}

func (r *SignKeyRepository) FindActive(ctx context.Context) (*model.SignKey, error) {
	var key model.SignKey
	result := r.db.WithContext(ctx).
		Where("active = true").
		Order("created_at DESC").
		First(&key)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &key, nil
}

func (r *SignKeyRepository) FindAllActive(ctx context.Context) ([]model.SignKey, error) {
	var keys []model.SignKey
	result := r.db.WithContext(ctx).
		Where("active = true").
		Order("created_at DESC").
		Find(&keys)
	if result.Error != nil {
		return nil, result.Error
	}
	return keys, nil
}

func (r *SignKeyRepository) FindByKID(ctx context.Context, kid string) (*model.SignKey, error) {
	var key model.SignKey
	result := r.db.WithContext(ctx).Where("kid = ?", kid).First(&key)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &key, nil
}

// FindAll は有効・無効を問わず全ての署名鍵を返す。
func (r *SignKeyRepository) FindAll(ctx context.Context) ([]model.SignKey, error) {
	var keys []model.SignKey
	result := r.db.WithContext(ctx).Order("created_at DESC").Find(&keys)
	if result.Error != nil {
		return nil, result.Error
	}
	return keys, nil
}

// Deactivate は鍵を無効化し、ローテーション日時を記録する。
func (r *SignKeyRepository) Deactivate(ctx context.Context, kid string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.SignKey{}).
		Where("kid = ?", kid).
		Updates(map[string]interface{}{"active": false, "rotated_at": now}).Error
}

// DeactivateAllActive は現在有効な全ての鍵を無効化する。
func (r *SignKeyRepository) DeactivateAllActive(ctx context.Context) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.SignKey{}).
		Where("active = true").
		Updates(map[string]interface{}{"active": false, "rotated_at": now}).Error
}
