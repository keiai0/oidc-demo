package model

import (
	"time"

	"github.com/google/uuid"
)

type AccessToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	JTI       string     `gorm:"type:varchar(255);uniqueIndex;not null"`
	SessionID uuid.UUID  `gorm:"type:uuid;not null"`
	ClientID  uuid.UUID  `gorm:"type:uuid;not null"`
	Scope     string     `gorm:"type:varchar(1024);not null"`
	ExpiresAt time.Time  `gorm:"not null"`
	RevokedAt *time.Time

	Session Session `gorm:"foreignKey:SessionID"`
	Client  Client  `gorm:"foreignKey:ClientID"`
}

func (AccessToken) TableName() string { return "access_tokens" }
