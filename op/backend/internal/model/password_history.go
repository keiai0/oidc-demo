package model

import (
	"time"

	"github.com/google/uuid"
)

type PasswordHistory struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID       uuid.UUID `gorm:"type:uuid;not null;index"`
	PasswordHash string    `gorm:"type:varchar(512);not null"`
	CreatedAt    time.Time
}

func (PasswordHistory) TableName() string { return "password_histories" }
