package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

// AdminUserRepository は管理者ユーザーの永続化操作を提供する。
type AdminUserRepository struct {
	db *gorm.DB
}

// NewAdminUserRepository は AdminUserRepository を生成する。
func NewAdminUserRepository(db *gorm.DB) *AdminUserRepository {
	return &AdminUserRepository{db: db}
}

// FindByLoginID はログイン ID で管理者ユーザーを検索する。見つからない場合は (nil, nil) を返す。
func (r *AdminUserRepository) FindByLoginID(ctx context.Context, loginID string) (*model.AdminUser, error) {
	var user model.AdminUser
	result := r.db.WithContext(ctx).First(&user, "login_id = ?", loginID)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

// FindByID は UUID で管理者ユーザーを検索する。見つからない場合は (nil, nil) を返す。
func (r *AdminUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.AdminUser, error) {
	var user model.AdminUser
	result := r.db.WithContext(ctx).First(&user, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

// UpdateLastLoginAt は最終ログイン日時を更新する。
func (r *AdminUserRepository) UpdateLastLoginAt(ctx context.Context, id uuid.UUID, t time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.AdminUser{}).
		Where("id = ?", id).
		Update("last_login_at", t).Error
}
