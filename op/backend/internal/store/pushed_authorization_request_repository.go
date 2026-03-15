package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

// PushedAuthorizationRequestRepository は PAR の永続化を担当する。
type PushedAuthorizationRequestRepository struct {
	db *gorm.DB
}

func NewPushedAuthorizationRequestRepository(db *gorm.DB) *PushedAuthorizationRequestRepository {
	return &PushedAuthorizationRequestRepository{db: db}
}

func (r *PushedAuthorizationRequestRepository) Create(ctx context.Context, par *model.PushedAuthorizationRequest) error {
	return r.db.WithContext(ctx).Create(par).Error
}

func (r *PushedAuthorizationRequestRepository) FindByRequestURI(ctx context.Context, requestURI string) (*model.PushedAuthorizationRequest, error) {
	var par model.PushedAuthorizationRequest
	result := r.db.WithContext(ctx).Where("request_uri = ?", requestURI).First(&par)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &par, nil
}

func (r *PushedAuthorizationRequestRepository) MarkAsUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.PushedAuthorizationRequest{}).
		Where("id = ?", id).
		Update("used_at", now).Error
}
