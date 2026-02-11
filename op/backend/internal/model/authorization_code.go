package model

import (
	"time"

	"github.com/google/uuid"
)

type AuthorizationCode struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	SessionID           uuid.UUID  `gorm:"type:uuid;not null"`
	ClientID            uuid.UUID  `gorm:"type:uuid;not null"`
	Code                string     `gorm:"type:varchar(255);uniqueIndex;not null"`
	RedirectURI         string     `gorm:"type:varchar(2048);not null"`
	Scope               string     `gorm:"type:varchar(1024);not null"`
	Nonce               *string    `gorm:"type:varchar(255)"`
	CodeChallenge       *string    `gorm:"type:varchar(255)"`
	CodeChallengeMethod *string    `gorm:"type:varchar(31)"`
	ExpiresAt           time.Time  `gorm:"not null"`
	UsedAt              *time.Time

	Session Session `gorm:"foreignKey:SessionID"`
	Client  Client  `gorm:"foreignKey:ClientID"`
}

func (AuthorizationCode) TableName() string { return "authorization_codes" }

func (ac *AuthorizationCode) IsUsed() bool {
	return ac.UsedAt != nil
}

func (ac *AuthorizationCode) IsExpired() bool {
	return ac.ExpiresAt.Before(time.Now())
}
