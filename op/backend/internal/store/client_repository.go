package store

import (
	"context"
	"errors"

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
