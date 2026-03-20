package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) FindByTenantAndLoginID(ctx context.Context, tenantID uuid.UUID, loginID string) (*model.User, error) {
	var user model.User
	result := r.db.WithContext(ctx).
		Preload("Credentials.PasswordCredential").
		Where("tenant_id = ? AND login_id = ?", tenantID, loginID).
		First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	result := r.db.WithContext(ctx).First(&user, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

func (r *UserRepository) FindByTenantAndEmail(ctx context.Context, tenantID uuid.UUID, email string) (*model.User, error) {
	var user model.User
	result := r.db.WithContext(ctx).
		Where("tenant_id = ? AND email = ?", tenantID, email).
		First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

func (r *UserRepository) FindByIDWithCredentials(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	result := r.db.WithContext(ctx).
		Preload("Credentials.PasswordCredential").
		First(&user, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

func (r *UserRepository) UpdateLastLoginAt(ctx context.Context, id uuid.UUID, t time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", id).
		Update("last_login_at", t).Error
}

// IncrementFailedLogin は連続失敗回数をインクリメントし、閾値に達したらロックする。
// ロックアウトパラメータ: 5回失敗 → 15分ロック
func (r *UserRepository) IncrementFailedLogin(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"failed_login_count": gorm.Expr("failed_login_count + 1"),
			"locked_until": gorm.Expr(
				"CASE WHEN failed_login_count + 1 >= 5 THEN NOW() + INTERVAL '15 minutes' ELSE locked_until END",
			),
		}).Error
}

// ResetFailedLogin は連続失敗回数とロック状態をリセットする。
func (r *UserRepository) ResetFailedLogin(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"failed_login_count": 0,
			"locked_until":      nil,
		}).Error
}

// FindByTenantAndExternalID は SCIM externalId でユーザーを検索する。
func (r *UserRepository) FindByTenantAndExternalID(ctx context.Context, tenantID uuid.UUID, externalID string) (*model.User, error) {
	var user model.User
	result := r.db.WithContext(ctx).
		Where("tenant_id = ? AND external_id = ?", tenantID, externalID).
		First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

// ListByTenantIDWithFilter は SCIM 用のフィルタ付きユーザー一覧を返す。
// filterAttr/filterValue が非空の場合、対応するカラムで eq フィルタを適用する。
func (r *UserRepository) ListByTenantIDWithFilter(ctx context.Context, tenantID uuid.UUID, filterAttr, filterValue string, offset, limit int) ([]model.User, int64, error) {
	query := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)

	if filterAttr != "" && filterValue != "" {
		switch filterAttr {
		case "login_id":
			query = query.Where("login_id = ?", filterValue)
		case "email":
			query = query.Where("email = ?", filterValue)
		case "external_id":
			query = query.Where("external_id = ?", filterValue)
		case "status":
			query = query.Where("status = ?", filterValue)
		}
	}

	var total int64
	if err := query.Model(&model.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []model.User
	if err := query.Order("created_at ASC").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// UpdateSCIM は SCIM PATCH 操作で変更されたユーザーフィールドを保存する。
func (r *UserRepository) UpdateSCIM(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Deactivate はユーザーを無効化する（ソフトデリート: status → disabled）。
func (r *UserRepository) Deactivate(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", id).
		Update("status", "disabled").Error
}

// UpdateEmail はユーザーのメールアドレスを更新し、email_verified を true にする。
// メールアドレス変更トークンの検証完了後に呼び出す。
func (r *UserRepository) UpdateEmail(ctx context.Context, id uuid.UUID, newEmail string) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"email":          newEmail,
			"email_verified": true,
		}).Error
}
