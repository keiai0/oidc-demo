package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

// EmailChangeTokenRepository はメールアドレス変更トークンのリポジトリ。
type EmailChangeTokenRepository struct {
	db *gorm.DB
}

func NewEmailChangeTokenRepository(db *gorm.DB) *EmailChangeTokenRepository {
	return &EmailChangeTokenRepository{db: db}
}

// Create はメールアドレス変更トークンを作成する。
func (r *EmailChangeTokenRepository) Create(ctx context.Context, token *model.EmailChangeToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

// FindByTokenHash はトークンハッシュでメールアドレス変更トークンを検索する。
// 未発見の場合は (nil, nil) を返す。
func (r *EmailChangeTokenRepository) FindByTokenHash(ctx context.Context, hash string) (*model.EmailChangeToken, error) {
	var token model.EmailChangeToken
	result := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&token)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &token, nil
}

// MarkAsUsed はトークンを使用済みにする。
func (r *EmailChangeTokenRepository) MarkAsUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.EmailChangeToken{}).
		Where("id = ?", id).
		Update("used_at", now).Error
}

// InvalidateByUserID はユーザーの未使用メールアドレス変更トークンを全て無効化する。
func (r *EmailChangeTokenRepository) InvalidateByUserID(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.EmailChangeToken{}).
		Where("user_id = ? AND used_at IS NULL", userID).
		Update("used_at", now).Error
}
