package store

import (
	"context"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

type IDTokenRepository struct {
	db *gorm.DB
}

func NewIDTokenRepository(db *gorm.DB) *IDTokenRepository {
	return &IDTokenRepository{db: db}
}

func (r *IDTokenRepository) Create(ctx context.Context, token *model.IDToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}
