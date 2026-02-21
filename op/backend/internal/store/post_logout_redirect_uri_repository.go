package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

// PostLogoutRedirectURIRepository はポストログアウトリダイレクト URI の永続化を処理する。
type PostLogoutRedirectURIRepository struct {
	db *gorm.DB
}

// NewPostLogoutRedirectURIRepository は PostLogoutRedirectURIRepository を生成する。
func NewPostLogoutRedirectURIRepository(db *gorm.DB) *PostLogoutRedirectURIRepository {
	return &PostLogoutRedirectURIRepository{db: db}
}

// ListByClientID はクライアントに属する全てのポストログアウトリダイレクト URI を返す。
func (r *PostLogoutRedirectURIRepository) ListByClientID(ctx context.Context, clientDBID uuid.UUID) ([]model.PostLogoutRedirectURI, error) {
	var uris []model.PostLogoutRedirectURI
	result := r.db.WithContext(ctx).
		Where("client_id = ?", clientDBID).
		Order("created_at ASC").
		Find(&uris)
	if result.Error != nil {
		return nil, result.Error
	}
	return uris, nil
}

// Create は新しいポストログアウトリダイレクト URI を永続化する。
func (r *PostLogoutRedirectURIRepository) Create(ctx context.Context, uri *model.PostLogoutRedirectURI) error {
	return r.db.WithContext(ctx).Create(uri).Error
}

// Delete は UUID でポストログアウトリダイレクト URI を削除する。
func (r *PostLogoutRedirectURIRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.PostLogoutRedirectURI{}, "id = ?", id).Error
}
