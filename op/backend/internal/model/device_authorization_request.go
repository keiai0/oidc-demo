package model

import (
	"time"

	"github.com/google/uuid"
)

// DeviceAuthorizationRequest は Device Authorization Grant (RFC 8628) のリクエスト情報を保持する。
type DeviceAuthorizationRequest struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID     uuid.UUID  `gorm:"type:uuid;not null"`
	ClientID     uuid.UUID  `gorm:"type:uuid;not null"`
	DeviceCode   string     `gorm:"type:varchar(255);uniqueIndex;not null"`
	UserCode     string     `gorm:"type:varchar(15);uniqueIndex;not null"`
	Scope        string     `gorm:"type:varchar(1024);not null;default:''"`
	Status       string     `gorm:"type:varchar(31);not null;default:'pending'"`
	SessionID    *uuid.UUID `gorm:"type:uuid"`
	PollInterval int        `gorm:"not null;default:5"`
	ExpiresAt    time.Time  `gorm:"not null"`
	LastPolledAt *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time

	Tenant  Tenant  `gorm:"foreignKey:TenantID"`
	Client  Client  `gorm:"foreignKey:ClientID"`
	Session *Session `gorm:"foreignKey:SessionID"`
}

func (DeviceAuthorizationRequest) TableName() string { return "device_authorization_requests" }

// IsExpired はリクエストが有効期限切れかどうかを返す。
func (d *DeviceAuthorizationRequest) IsExpired() bool {
	return d.ExpiresAt.Before(time.Now())
}

// IsPending はリクエストがユーザーの承認待ちかどうかを返す。
func (d *DeviceAuthorizationRequest) IsPending() bool {
	return d.Status == "pending"
}

// IsApproved はリクエストがユーザーに承認されたかどうかを返す。
func (d *DeviceAuthorizationRequest) IsApproved() bool {
	return d.Status == "approved"
}

// IsDenied はリクエストがユーザーに拒否されたかどうかを返す。
func (d *DeviceAuthorizationRequest) IsDenied() bool {
	return d.Status == "denied"
}
