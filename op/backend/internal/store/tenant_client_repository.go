package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

// TenantClientRepository はテナント-クライアント中間テーブルの永続化操作を提供する。
type TenantClientRepository struct {
	db *gorm.DB
}

// NewTenantClientRepository は TenantClientRepository を生成する。
func NewTenantClientRepository(db *gorm.DB) *TenantClientRepository {
	return &TenantClientRepository{db: db}
}

// ExistsByTenantAndClient はテナントでクライアントが利用可能か確認する。
func (r *TenantClientRepository) ExistsByTenantAndClient(ctx context.Context, tenantID, clientID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.TenantClient{}).
		Where("tenant_id = ? AND client_id = ? AND enabled = true", tenantID, clientID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Create は中間テーブルにレコードを追加する。
func (r *TenantClientRepository) Create(ctx context.Context, tc *model.TenantClient) error {
	return r.db.WithContext(ctx).Create(tc).Error
}

// Delete はテナント-クライアント紐づけを削除する。
func (r *TenantClientRepository) Delete(ctx context.Context, tenantID, clientID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("tenant_id = ? AND client_id = ?", tenantID, clientID).
		Delete(&model.TenantClient{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ListByClientID はクライアントに紐づくテナント-クライアント関連を返す。
func (r *TenantClientRepository) ListByClientID(ctx context.Context, clientID uuid.UUID) ([]model.TenantClient, error) {
	var tcs []model.TenantClient
	err := r.db.WithContext(ctx).
		Preload("Tenant").
		Where("client_id = ?", clientID).
		Order("created_at DESC").
		Find(&tcs).Error
	if err != nil {
		return nil, err
	}
	return tcs, nil
}
