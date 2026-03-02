package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// TotpConfigRepository は totp_configs テーブルの GORM 実装。
type TotpConfigRepository struct {
	db *gorm.DB
}

// NewTotpConfigRepository は TotpConfigRepository を生成する。
func NewTotpConfigRepository(db *gorm.DB) *TotpConfigRepository {
	return &TotpConfigRepository{db: db}
}

// Create は TOTP 設定を作成する。
func (r *TotpConfigRepository) Create(ctx context.Context, config *model.TotpConfig) error {
	if err := r.db.WithContext(ctx).Create(config).Error; err != nil {
		return fmt.Errorf("failed to create TOTP config: %w", err)
	}
	return nil
}

// FindByMfaConfigID は MFA 設定 ID から TOTP 設定を取得する。
func (r *TotpConfigRepository) FindByMfaConfigID(ctx context.Context, mfaConfigID uuid.UUID) (*model.TotpConfig, error) {
	var config model.TotpConfig
	err := r.db.WithContext(ctx).
		Where("mfa_config_id = ?", mfaConfigID).
		First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find TOTP config: %w", err)
	}
	return &config, nil
}

// UpdateLastUsedStep は最終使用ステップ番号を更新する（リプレイ攻撃防止）。
func (r *TotpConfigRepository) UpdateLastUsedStep(ctx context.Context, id uuid.UUID, step int64) error {
	err := r.db.WithContext(ctx).
		Model(&model.TotpConfig{}).
		Where("id = ?", id).
		Update("last_used_step", step).Error
	if err != nil {
		return fmt.Errorf("failed to update last used step: %w", err)
	}
	return nil
}

// Delete は TOTP 設定を削除する。
func (r *TotpConfigRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&model.TotpConfig{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete TOTP config: %w", err)
	}
	return nil
}
