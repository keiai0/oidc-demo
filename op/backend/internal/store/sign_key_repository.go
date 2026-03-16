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

// FindActive は status = 'active' の最新鍵を返す。
func (r *SignKeyRepository) FindActive(ctx context.Context) (*model.SignKey, error) {
	var key model.SignKey
	result := r.db.WithContext(ctx).
		Where("status = ?", model.SignKeyStatusActive).
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

// FindAllByStatus は指定ステータスの全鍵を返す。
func (r *SignKeyRepository) FindAllByStatus(ctx context.Context, status string) ([]model.SignKey, error) {
	var keys []model.SignKey
	result := r.db.WithContext(ctx).
		Where("status = ?", status).
		Order("created_at DESC").
		Find(&keys)
	if result.Error != nil {
		return nil, result.Error
	}
	return keys, nil
}

// UpdateStatus は指定 kid の鍵ステータスと rotated_at を更新する。
func (r *SignKeyRepository) UpdateStatus(ctx context.Context, kid string, status string) error {
	updates := map[string]interface{}{"status": status}
	if status == model.SignKeyStatusPassive || status == model.SignKeyStatusExpired {
		now := time.Now()
		updates["rotated_at"] = now
	}
	return r.db.WithContext(ctx).
		Model(&model.SignKey{}).
		Where("kid = ?", kid).
		Updates(updates).Error
}

// UpdateStatusBulk は fromStatus の全鍵を toStatus に一括更新する。
func (r *SignKeyRepository) UpdateStatusBulk(ctx context.Context, fromStatus string, toStatus string) error {
	updates := map[string]interface{}{"status": toStatus}
	if toStatus == model.SignKeyStatusPassive || toStatus == model.SignKeyStatusExpired {
		now := time.Now()
		updates["rotated_at"] = now
	}
	return r.db.WithContext(ctx).
		Model(&model.SignKey{}).
		Where("status = ?", fromStatus).
		Updates(updates).Error
}

// DeleteByStatus は指定ステータスの全鍵を削除する。
func (r *SignKeyRepository) DeleteByStatus(ctx context.Context, status string) error {
	return r.db.WithContext(ctx).
		Where("status = ?", status).
		Delete(&model.SignKey{}).Error
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

// FindAll は全ての署名鍵を返す。
func (r *SignKeyRepository) FindAll(ctx context.Context) ([]model.SignKey, error) {
	var keys []model.SignKey
	result := r.db.WithContext(ctx).Order("created_at DESC").Find(&keys)
	if result.Error != nil {
		return nil, result.Error
	}
	return keys, nil
}
