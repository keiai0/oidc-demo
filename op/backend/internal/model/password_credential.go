package model

import (
	"time"

	"github.com/google/uuid"
)

type PasswordCredential struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CredentialID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	PasswordHash string    `gorm:"type:varchar(512);not null"`
	Algorithm    string    `gorm:"type:varchar(63);not null;default:'argon2id'"`
	UpdatedAt    time.Time
}

func (PasswordCredential) TableName() string { return "password_credentials" }
