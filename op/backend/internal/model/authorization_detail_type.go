package model

import (
	"time"

	"github.com/google/uuid"
)

// AuthorizationDetailType は Rich Authorization Requests (RFC 9396) の認可詳細タイプ定義。
// テナントごとにサポートする type を登録する。
type AuthorizationDetailType struct {
	ID               uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID         uuid.UUID   `gorm:"type:uuid;not null"`
	TypeName         string      `gorm:"type:varchar(255);not null"`
	Description      *string     `gorm:"type:text"`
	JSONSchema       *string     `gorm:"type:jsonb;column:json_schema"`
	AllowedActions   StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	AllowedLocations StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	CreatedAt        time.Time
	UpdatedAt        time.Time

	Tenant Tenant `gorm:"foreignKey:TenantID"`
}

func (AuthorizationDetailType) TableName() string { return "authorization_detail_types" }
