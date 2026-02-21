package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

// RedirectURIRepository はリダイレクト URI の永続化を処理する。
type RedirectURIRepository struct {
	db *gorm.DB
}

// NewRedirectURIRepository は RedirectURIRepository を生成する。
func NewRedirectURIRepository(db *gorm.DB) *RedirectURIRepository {
	return &RedirectURIRepository{db: db}
}

// ListByClientID はクライアントに属する全てのリダイレクト URI を返す。
func (r *RedirectURIRepository) ListByClientID(ctx context.Context, clientDBID uuid.UUID) ([]model.RedirectURI, error) {
	var uris []model.RedirectURI
	result := r.db.WithContext(ctx).
		Where("client_id = ?", clientDBID).
		Order("created_at ASC").
		Find(&uris)
	if result.Error != nil {
		return nil, result.Error
	}
	return uris, nil
}

// Create は新しいリダイレクト URI を永続化する。
func (r *RedirectURIRepository) Create(ctx context.Context, uri *model.RedirectURI) error {
	return r.db.WithContext(ctx).Create(uri).Error
}

// FindByID は UUID でリダイレクト URI を検索する。
func (r *RedirectURIRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.RedirectURI, error) {
	var uri model.RedirectURI
	result := r.db.WithContext(ctx).First(&uri, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &uri, nil
}

// Delete は UUID でリダイレクト URI を削除する。
func (r *RedirectURIRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.RedirectURI{}, "id = ?", id).Error
}
