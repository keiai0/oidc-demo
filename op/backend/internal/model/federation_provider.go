package model

import (
	"time"

	"github.com/google/uuid"
)

// FederationProvider は外部 IdP 連携プロバイダの設定を保持する。
type FederationProvider struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID        uuid.UUID `gorm:"type:uuid;not null"`
	Name            string    `gorm:"type:varchar(63);not null"`
	Issuer          string    `gorm:"type:varchar(2048);not null"`
	ClientID        string    `gorm:"type:varchar(255);not null"`
	ClientSecretEnc string    `gorm:"type:text;not null"`
	Scopes          string    `gorm:"type:varchar(1024);not null;default:'openid profile email'"`
	AutoProvision   bool      `gorm:"not null;default:true"`
	Status          string    `gorm:"type:varchar(31);not null;default:'active'"`
	CreatedAt       time.Time
	UpdatedAt       time.Time

	Tenant Tenant `gorm:"foreignKey:TenantID"`
}

func (FederationProvider) TableName() string { return "federation_providers" }

// IsActive はプロバイダが有効かどうかを返す。
func (f *FederationProvider) IsActive() bool {
	return f.Status == "active"
}
