package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

// AdminSessionRepository は管理者セッションの永続化操作を提供する。
type AdminSessionRepository struct {
	db *gorm.DB
}

// NewAdminSessionRepository は AdminSessionRepository を生成する。
func NewAdminSessionRepository(db *gorm.DB) *AdminSessionRepository {
	return &AdminSessionRepository{db: db}
}

// Create は新しい管理者セッションを永続化する。
func (r *AdminSessionRepository) Create(ctx context.Context, session *model.AdminSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

// FindByID は UUID で管理者セッションを検索する。見つからない場合は (nil, nil) を返す。
func (r *AdminSessionRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.AdminSession, error) {
	var session model.AdminSession
	result := r.db.WithContext(ctx).First(&session, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &session, nil
}

// Revoke は管理者セッションを失効済みにする。
func (r *AdminSessionRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.AdminSession{}).
		Where("id = ?", id).
		Update("revoked_at", now).Error
}
