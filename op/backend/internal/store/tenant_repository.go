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
