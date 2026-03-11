package model

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID   `gorm:"type:uuid;not null"`
	TenantID  uuid.UUID   `gorm:"type:uuid;not null"`
	IPAddress string      `gorm:"type:varchar(45);not null"`
	UserAgent string      `gorm:"type:text;not null;default:''"`
	AuthTime  time.Time   `gorm:"not null;default:now()"`
	AMR       StringSlice `gorm:"type:jsonb;not null;default:'[\"pwd\"]'"`
	ACR       string      `gorm:"type:varchar(255);not null;default:'urn:mace:incommon:iap:bronze'"`
	PendingMFA       bool        `gorm:"not null;default:false"`
	MfaSetupRequired bool        `gorm:"not null;default:false"`
	ExpiresAt        time.Time   `gorm:"not null"`
	RevokedAt  *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time

	User   User   `gorm:"foreignKey:UserID"`
	Tenant Tenant `gorm:"foreignKey:TenantID"`
}

func (Session) TableName() string { return "sessions" }

func (s *Session) IsValid() bool {
	return s.RevokedAt == nil && s.ExpiresAt.After(time.Now())
}

// IsFullyAuthenticated はセッションが有効かつ MFA 検証が完了しているかを返す。
func (s *Session) IsFullyAuthenticated() bool {
	return s.IsValid() && !s.PendingMFA
}
