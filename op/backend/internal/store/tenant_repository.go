package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

type TenantRepository struct {
	db *gorm.DB
}

func NewTenantRepository(db *gorm.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

func (r *TenantRepository) FindByCode(ctx context.Context, code string) (*model.Tenant, error) {
	var tenant model.Tenant
	result := r.db.WithContext(ctx).Where("code = ?", code).First(&tenant)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &tenant, nil
}

func (r *TenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	var tenant model.Tenant
	result := r.db.WithContext(ctx).First(&tenant, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &tenant, nil
}

// List はテナントをページネーション付きで返す。
func (r *TenantRepository) List(ctx context.Context, limit, offset int) ([]model.Tenant, int64, error) {
	var tenants []model.Tenant
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Tenant{}).Count(&count).Error; err != nil {
		return nil, 0, err
	}
	result := r.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Offset(offset).Find(&tenants)
	if result.Error != nil {
		return nil, 0, result.Error
	}
	return tenants, count, nil
}

// Create は新しいテナントを永続化する。
func (r *TenantRepository) Create(ctx context.Context, tenant *model.Tenant) error {
	return r.db.WithContext(ctx).Create(tenant).Error
}

// Update はテナントの変更を保存する。
func (r *TenantRepository) Update(ctx context.Context, tenant *model.Tenant) error {
	return r.db.WithContext(ctx).Save(tenant).Error
}
