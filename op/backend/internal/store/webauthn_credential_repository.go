package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// WebAuthnCredentialRepository は webauthn_credentials テーブルの GORM 実装。
type WebAuthnCredentialRepository struct {
	db *gorm.DB
}

// NewWebAuthnCredentialRepository は WebAuthnCredentialRepository を生成する。
func NewWebAuthnCredentialRepository(db *gorm.DB) *WebAuthnCredentialRepository {
	return &WebAuthnCredentialRepository{db: db}
}

// Create は WebAuthn クレデンシャルを作成する。
func (r *WebAuthnCredentialRepository) Create(ctx context.Context, cred *model.WebAuthnCredential) error {
	if err := r.db.WithContext(ctx).Create(cred).Error; err != nil {
		return fmt.Errorf("failed to create WebAuthn credential: %w", err)
	}
	return nil
}

// FindByMfaConfigID は指定 MFA 設定の全 WebAuthn クレデンシャルを返す。
func (r *WebAuthnCredentialRepository) FindByMfaConfigID(ctx context.Context, mfaConfigID uuid.UUID) ([]model.WebAuthnCredential, error) {
	var creds []model.WebAuthnCredential
	if err := r.db.WithContext(ctx).Where("mfa_config_id = ?", mfaConfigID).Find(&creds).Error; err != nil {
		return nil, fmt.Errorf("failed to find WebAuthn credentials: %w", err)
	}
	return creds, nil
}

// FindByCredentialID は credential_id で WebAuthn クレデンシャルを検索する。
func (r *WebAuthnCredentialRepository) FindByCredentialID(ctx context.Context, credentialID string) (*model.WebAuthnCredential, error) {
	var cred model.WebAuthnCredential
	err := r.db.WithContext(ctx).Where("credential_id = ?", credentialID).First(&cred).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find WebAuthn credential: %w", err)
	}
	return &cred, nil
}

// UpdateSignCount は sign_count を更新する。
func (r *WebAuthnCredentialRepository) UpdateSignCount(ctx context.Context, id uuid.UUID, signCount uint32) error {
	if err := r.db.WithContext(ctx).Model(&model.WebAuthnCredential{}).Where("id = ?", id).Update("sign_count", signCount).Error; err != nil {
		return fmt.Errorf("failed to update sign count: %w", err)
	}
	return nil
}

// Delete は WebAuthn クレデンシャルを削除する。
func (r *WebAuthnCredentialRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&model.WebAuthnCredential{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete WebAuthn credential: %w", err)
	}
	return nil
}
