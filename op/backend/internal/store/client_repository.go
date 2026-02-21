package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

type ClientRepository struct {
	db *gorm.DB
}

func NewClientRepository(db *gorm.DB) *ClientRepository {
	return &ClientRepository{db: db}
}

func (r *ClientRepository) FindByClientID(ctx context.Context, clientID string) (*model.Client, error) {
	var client model.Client
	result := r.db.WithContext(ctx).Where("client_id = ?", clientID).First(&client)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &client, nil
}

func (r *ClientRepository) FindByClientIDWithRedirectURIs(ctx context.Context, clientID string) (*model.Client, error) {
	var client model.Client
	result := r.db.WithContext(ctx).
		Preload("RedirectURIs").
		Preload("PostLogoutRedirectURIs").
		Where("client_id = ?", clientID).
		First(&client)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &client, nil
}

// ListByTenantID はテナントに属するクライアントをページネーション付きで返す。
func (r *ClientRepository) ListByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]model.Client, int64, error) {
	var clients []model.Client
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Client{}).Where("tenant_id = ?", tenantID).Count(&count).Error; err != nil {
		return nil, 0, err
	}
	result := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&clients)
	if result.Error != nil {
		return nil, 0, result.Error
	}
	return clients, count, nil
}

// FindByID は UUID でクライアントを検索する。
func (r *ClientRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Client, error) {
	var client model.Client
	result := r.db.WithContext(ctx).First(&client, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &client, nil
}

// FindByIDWithRelations は UUID でクライアントを検索し、リダイレクト URI をプリロードして返す。
func (r *ClientRepository) FindByIDWithRelations(ctx context.Context, id uuid.UUID) (*model.Client, error) {
	var client model.Client
	result := r.db.WithContext(ctx).
		Preload("RedirectURIs").
		Preload("PostLogoutRedirectURIs").
		First(&client, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &client, nil
}

// Create は新しいクライアントを永続化する。
func (r *ClientRepository) Create(ctx context.Context, client *model.Client) error {
	return r.db.WithContext(ctx).Create(client).Error
}

// Update はクライアントの変更を保存する。
func (r *ClientRepository) Update(ctx context.Context, client *model.Client) error {
	return r.db.WithContext(ctx).Save(client).Error
}

// UpdateSecretHash は client_secret_hash フィールドのみを更新する。
func (r *ClientRepository) UpdateSecretHash(ctx context.Context, id uuid.UUID, hash string) error {
	return r.db.WithContext(ctx).
		Model(&model.Client{}).
		Where("id = ?", id).
		Update("client_secret_hash", hash).Error
}

// SoftDelete はクライアントの status を "disabled" に設定して論理削除する。
func (r *ClientRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.Client{}).
		Where("id = ?", id).
		Update("status", "disabled").Error
}
