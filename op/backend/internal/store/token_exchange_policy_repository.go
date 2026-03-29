package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// TokenExchangePolicyRepository は Token Exchange ポリシーの永続化操作を実装する。
type TokenExchangePolicyRepository struct {
	db *gorm.DB
}

// NewTokenExchangePolicyRepository は TokenExchangePolicyRepository を生成する。
func NewTokenExchangePolicyRepository(db *gorm.DB) *TokenExchangePolicyRepository {
	return &TokenExchangePolicyRepository{db: db}
}

// Create はポリシーを作成する。
func (r *TokenExchangePolicyRepository) Create(ctx context.Context, policy *model.TokenExchangePolicy) error {
	return r.db.WithContext(ctx).Create(policy).Error
}

// FindByClientID はクライアント DB ID でポリシーを検索する。見つからなければ (nil, nil)。
func (r *TokenExchangePolicyRepository) FindByClientID(ctx context.Context, clientID uuid.UUID) (*model.TokenExchangePolicy, error) {
	var policy model.TokenExchangePolicy
	result := r.db.WithContext(ctx).
		Where("client_id = ?", clientID).
		First(&policy)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &policy, nil
}

// Update はポリシーを更新する。
func (r *TokenExchangePolicyRepository) Update(ctx context.Context, policy *model.TokenExchangePolicy) error {
	return r.db.WithContext(ctx).Save(policy).Error
}

// Delete はポリシーを削除する。
func (r *TokenExchangePolicyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.TokenExchangePolicy{}, id).Error
}
