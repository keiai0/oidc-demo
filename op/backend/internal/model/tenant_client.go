package model

import (
	"time"

	"github.com/google/uuid"
)

// TenantClient はテナントとクライアントの多対多関連を表す。
type TenantClient struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID  uuid.UUID `gorm:"type:uuid;not null"`
	ClientID  uuid.UUID `gorm:"type:uuid;not null"`
	Enabled   bool      `gorm:"not null;default:true"`
	CreatedAt time.Time

	Tenant Tenant `gorm:"foreignKey:TenantID"`
	Client Client `gorm:"foreignKey:ClientID"`
}

func (TenantClient) TableName() string { return "tenant_clients" }
