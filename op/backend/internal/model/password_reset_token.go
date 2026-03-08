package model

import (
	"time"

	"github.com/google/uuid"
)

type PasswordResetToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	TokenHash string     `gorm:"type:varchar(512);not null;uniqueIndex"`
	ExpiresAt time.Time  `gorm:"not null"`
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (PasswordResetToken) TableName() string { return "password_reset_tokens" }

// IsExpired はトークンが有効期限切れかを返す。
func (t *PasswordResetToken) IsExpired() bool {
	return t.ExpiresAt.Before(time.Now())
}

// IsUsed はトークンが使用済みかを返す。
func (t *PasswordResetToken) IsUsed() bool {
	return t.UsedAt != nil
}
