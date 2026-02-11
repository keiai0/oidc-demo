package model

import (
	"time"

	"github.com/google/uuid"
)

type IDToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	JTI       string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	SessionID uuid.UUID `gorm:"type:uuid;not null"`
	ClientID  uuid.UUID `gorm:"type:uuid;not null"`
	Nonce     *string   `gorm:"type:varchar(255)"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}

func (IDToken) TableName() string { return "id_tokens" }
