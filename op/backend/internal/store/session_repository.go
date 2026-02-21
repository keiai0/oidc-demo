package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

type SessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, session *model.Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *SessionRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Session, error) {
	var session model.Session
	result := r.db.WithContext(ctx).First(&session, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &session, nil
}

func (r *SessionRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.Session{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now).Error
}

// RevokeAll は全ての有効なセッションを失効させる。影響行数を返す。
func (r *SessionRepository) RevokeAll(ctx context.Context) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.Session{}).
		Where("revoked_at IS NULL").
		Update("revoked_at", now)
	return result.RowsAffected, result.Error
}

// RevokeByTenantID はテナントに属する全ての有効なセッションを失効させる。
func (r *SessionRepository) RevokeByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.Session{}).
		Where("tenant_id = ? AND revoked_at IS NULL", tenantID).
		Update("revoked_at", now)
	return result.RowsAffected, result.Error
}

// RevokeByUserID はユーザーに属する全ての有効なセッションを失効させる。
func (r *SessionRepository) RevokeByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.Session{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now)
	return result.RowsAffected, result.Error
}
