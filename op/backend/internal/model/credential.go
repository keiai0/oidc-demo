package model

import (
	"time"

	"github.com/google/uuid"
)

type Credential struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Type      string    `gorm:"type:varchar(63);not null"`
	CreatedAt time.Time
	UpdatedAt time.Time

	PasswordCredential *PasswordCredential `gorm:"foreignKey:CredentialID"`
}

func (Credential) TableName() string { return "credentials" }
