package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

type PasswordHistoryRepository struct {
	db *gorm.DB
}

func NewPasswordHistoryRepository(db *gorm.DB) *PasswordHistoryRepository {
	return &PasswordHistoryRepository{db: db}
}

// Create はパスワード履歴レコードを作成する。
func (r *PasswordHistoryRepository) Create(ctx context.Context, history *model.PasswordHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}

// FindRecentByUserID はユーザーの直近 limit 件のパスワード履歴を返す。
func (r *PasswordHistoryRepository) FindRecentByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]model.PasswordHistory, error) {
	var histories []model.PasswordHistory
	result := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&histories)
	if result.Error != nil {
		return nil, result.Error
	}
	return histories, nil
}
