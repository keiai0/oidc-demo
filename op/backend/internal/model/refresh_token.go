package model

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TokenHash       string     `gorm:"type:varchar(255);uniqueIndex;not null"`
	ParentID        *uuid.UUID `gorm:"type:uuid"`
	SessionID       uuid.UUID  `gorm:"type:uuid;not null;index"`
	AccessTokenID   uuid.UUID  `gorm:"type:uuid;not null"`
	ExpiresAt       time.Time  `gorm:"not null"`
	RevokedAt       *time.Time
	ReuseDetectedAt *time.Time

	Parent      *RefreshToken `gorm:"foreignKey:ParentID"`
	Session     Session       `gorm:"foreignKey:SessionID"`
	AccessToken AccessToken   `gorm:"foreignKey:AccessTokenID"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }
