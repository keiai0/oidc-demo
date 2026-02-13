package store

import (
	"context"
	"errors"

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
