package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// AuthorizationDetailTypeRepository は Rich Authorization Requests (RFC 9396) の認可詳細タイプの永続化操作を実装する。
type AuthorizationDetailTypeRepository struct {
	db *gorm.DB
}

// NewAuthorizationDetailTypeRepository は AuthorizationDetailTypeRepository を生成する。
func NewAuthorizationDetailTypeRepository(db *gorm.DB) *AuthorizationDetailTypeRepository {
	return &AuthorizationDetailTypeRepository{db: db}
}

// Create は認可詳細タイプを作成する。
func (r *AuthorizationDetailTypeRepository) Create(ctx context.Context, adt *model.AuthorizationDetailType) error {
	return r.db.WithContext(ctx).Create(adt).Error
}

// FindByID は UUID で認可詳細タイプを検索する。見つからなければ (nil, nil)。
func (r *AuthorizationDetailTypeRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.AuthorizationDetailType, error) {
	var adt model.AuthorizationDetailType
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&adt)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &adt, nil
}

// FindByTenantIDAndType はテナント ID とタイプ名で認可詳細タイプを検索する。見つからなければ (nil, nil)。
func (r *AuthorizationDetailTypeRepository) FindByTenantIDAndType(ctx context.Context, tenantID uuid.UUID, typeName string) (*model.AuthorizationDetailType, error) {
	var adt model.AuthorizationDetailType
	result := r.db.WithContext(ctx).
		Where("tenant_id = ? AND type_name = ?", tenantID, typeName).
		First(&adt)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &adt, nil
}

// ListByTenantID はテナントに属する認可詳細タイプの一覧を返す。
func (r *AuthorizationDetailTypeRepository) ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]model.AuthorizationDetailType, error) {
	var types []model.AuthorizationDetailType
	result := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at ASC").
		Find(&types)
	if result.Error != nil {
		return nil, result.Error
	}
	return types, nil
}

// Update は認可詳細タイプを更新する。
func (r *AuthorizationDetailTypeRepository) Update(ctx context.Context, adt *model.AuthorizationDetailType) error {
	return r.db.WithContext(ctx).Save(adt).Error
}

// Delete は認可詳細タイプを削除する。
func (r *AuthorizationDetailTypeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.AuthorizationDetailType{}, id).Error
}
