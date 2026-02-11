package model

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null"`
	TenantID  uuid.UUID  `gorm:"type:uuid;not null"`
	IPAddress string     `gorm:"type:varchar(45);not null"`
	UserAgent string     `gorm:"type:text;not null;default:''"`
	ExpiresAt time.Time  `gorm:"not null"`
	RevokedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time

	User   User   `gorm:"foreignKey:UserID"`
	Tenant Tenant `gorm:"foreignKey:TenantID"`
}

func (Session) TableName() string { return "sessions" }

func (s *Session) IsValid() bool {
	return s.RevokedAt == nil && s.ExpiresAt.After(time.Now())
}
