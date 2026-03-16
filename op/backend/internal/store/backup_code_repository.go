package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

// BackupCodeRepository は MFA バックアップコードのリポジトリ。
type BackupCodeRepository struct {
	db *gorm.DB
}

func NewBackupCodeRepository(db *gorm.DB) *BackupCodeRepository {
	return &BackupCodeRepository{db: db}
}

// Create はバックアップコードを作成する。
func (r *BackupCodeRepository) Create(ctx context.Context, code *model.BackupCode) error {
	return r.db.WithContext(ctx).Create(code).Error
}

// FindUnusedByUserID はユーザーの未使用バックアップコード一覧を返す。
// 検証時にメモリ上で argon2id 照合するため一括取得する。
func (r *BackupCodeRepository) FindUnusedByUserID(ctx context.Context, userID uuid.UUID) ([]model.BackupCode, error) {
	var codes []model.BackupCode
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND used_at IS NULL", userID).
		Find(&codes)
	return codes, result.Error
}

// MarkAsUsed はバックアップコードを使用済みにする。
func (r *BackupCodeRepository) MarkAsUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.BackupCode{}).
		Where("id = ?", id).
		Update("used_at", now).Error
}

// DeleteByUserID はユーザーのバックアップコードを全て削除する。
// 新規生成時に旧コードを消去するために使用する。
func (r *BackupCodeRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&model.BackupCode{}).Error
}
