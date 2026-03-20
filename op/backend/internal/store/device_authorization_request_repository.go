package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
	"gorm.io/gorm"
)

// DeviceAuthorizationRequestRepository は Device Authorization Grant (RFC 8628) リクエストの永続化を担当する。
type DeviceAuthorizationRequestRepository struct {
	db *gorm.DB
}

func NewDeviceAuthorizationRequestRepository(db *gorm.DB) *DeviceAuthorizationRequestRepository {
	return &DeviceAuthorizationRequestRepository{db: db}
}

func (r *DeviceAuthorizationRequestRepository) Create(ctx context.Context, req *model.DeviceAuthorizationRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *DeviceAuthorizationRequestRepository) FindByDeviceCode(ctx context.Context, deviceCode string) (*model.DeviceAuthorizationRequest, error) {
	var req model.DeviceAuthorizationRequest
	result := r.db.WithContext(ctx).Where("device_code = ?", deviceCode).First(&req)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &req, nil
}

func (r *DeviceAuthorizationRequestRepository) FindByUserCode(ctx context.Context, userCode string) (*model.DeviceAuthorizationRequest, error) {
	var req model.DeviceAuthorizationRequest
	result := r.db.WithContext(ctx).
		Preload("Client").
		Where("user_code = ?", userCode).
		First(&req)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &req, nil
}

// UpdateStatus はリクエストのステータスを更新する。承認時は session_id もセットする。
func (r *DeviceAuthorizationRequestRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, sessionID *uuid.UUID) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if sessionID != nil {
		updates["session_id"] = sessionID
	}
	return r.db.WithContext(ctx).
		Model(&model.DeviceAuthorizationRequest{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdateLastPolledAt はポーリング時刻を更新する。
func (r *DeviceAuthorizationRequestRepository) UpdateLastPolledAt(ctx context.Context, id uuid.UUID, t time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.DeviceAuthorizationRequest{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_polled_at": t,
			"updated_at":     t,
		}).Error
}

// IncrementPollInterval はポーリング間隔を指定秒数だけ増加させる (slow_down 応答用)。
func (r *DeviceAuthorizationRequestRepository) IncrementPollInterval(ctx context.Context, id uuid.UUID, incrementSec int) error {
	return r.db.WithContext(ctx).
		Model(&model.DeviceAuthorizationRequest{}).
		Where("id = ?", id).
		Update("poll_interval", gorm.Expr("poll_interval + ?", incrementSec)).Error
}
