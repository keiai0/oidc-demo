package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

// FederationProviderRepository は外部 IdP 連携プロバイダの永続化を担当する。
type FederationProviderRepository struct {
	db *gorm.DB
}

func NewFederationProviderRepository(db *gorm.DB) *FederationProviderRepository {
	return &FederationProviderRepository{db: db}
}

func (r *FederationProviderRepository) Create(ctx context.Context, provider *model.FederationProvider) error {
	return r.db.WithContext(ctx).Create(provider).Error
}

func (r *FederationProviderRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.FederationProvider, error) {
	var provider model.FederationProvider
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&provider)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &provider, nil
}

func (r *FederationProviderRepository) FindByTenantAndName(ctx context.Context, tenantID uuid.UUID, name string) (*model.FederationProvider, error) {
	var provider model.FederationProvider
	result := r.db.WithContext(ctx).
		Where("tenant_id = ? AND name = ?", tenantID, name).
		First(&provider)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &provider, nil
}

func (r *FederationProviderRepository) ListByTenantID(ctx context.Context, tenantID uuid.UUID) ([]model.FederationProvider, error) {
	var providers []model.FederationProvider
	result := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status = 'active'", tenantID).
		Order("name ASC").
		Find(&providers)
	if result.Error != nil {
		return nil, result.Error
	}
	return providers, nil
}

func (r *FederationProviderRepository) Update(ctx context.Context, provider *model.FederationProvider) error {
	return r.db.WithContext(ctx).Save(provider).Error
}

func (r *FederationProviderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.FederationProvider{}, "id = ?", id).Error
}
