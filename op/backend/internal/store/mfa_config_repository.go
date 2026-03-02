package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// MfaConfigRepository は mfa_configs テーブルの GORM 実装。
type MfaConfigRepository struct {
	db *gorm.DB
}

// NewMfaConfigRepository は MfaConfigRepository を生成する。
func NewMfaConfigRepository(db *gorm.DB) *MfaConfigRepository {
	return &MfaConfigRepository{db: db}
}

// FindEnabledByUserID はユーザーの有効な MFA 設定を TotpConfig 付きで返す。
func (r *MfaConfigRepository) FindEnabledByUserID(ctx context.Context, userID uuid.UUID) (*model.MfaConfig, error) {
	var config model.MfaConfig
	err := r.db.WithContext(ctx).
		Preload("TotpConfig").
		Where("user_id = ? AND enabled = true", userID).
		First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find enabled MFA config: %w", err)
	}
	return &config, nil
}

// FindByUserIDAndType はユーザーの指定タイプの MFA 設定を TotpConfig 付きで返す。
func (r *MfaConfigRepository) FindByUserIDAndType(ctx context.Context, userID uuid.UUID, mfaType string) (*model.MfaConfig, error) {
	var config model.MfaConfig
	err := r.db.WithContext(ctx).
		Preload("TotpConfig").
		Where("user_id = ? AND type = ?", userID, mfaType).
		First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find MFA config: %w", err)
	}
	return &config, nil
}

// Create は MFA 設定を作成する。
func (r *MfaConfigRepository) Create(ctx context.Context, config *model.MfaConfig) error {
	if err := r.db.WithContext(ctx).Create(config).Error; err != nil {
		return fmt.Errorf("failed to create MFA config: %w", err)
	}
	return nil
}

// Update は MFA 設定を更新する。
func (r *MfaConfigRepository) Update(ctx context.Context, config *model.MfaConfig) error {
	if err := r.db.WithContext(ctx).Save(config).Error; err != nil {
		return fmt.Errorf("failed to update MFA config: %w", err)
	}
	return nil
}

// Delete は MFA 設定を削除する（CASCADE で TotpConfig も削除される）。
func (r *MfaConfigRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&model.MfaConfig{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete MFA config: %w", err)
	}
	return nil
}
