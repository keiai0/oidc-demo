package model

import (
	"time"

	"github.com/google/uuid"
)

// EmailChangeToken はメールアドレス変更の確認トークンを表す。
// トークン平文はメールで送信し、DB には SHA-256 ハッシュのみ保存する。
type EmailChangeToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	NewEmail  string     `gorm:"type:varchar(255);not null"`
	TokenHash string     `gorm:"type:varchar(512);not null;uniqueIndex"`
	ExpiresAt time.Time  `gorm:"not null"`
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (EmailChangeToken) TableName() string { return "email_change_tokens" }

// IsExpired はトークンが有効期限切れかを返す。
func (t *EmailChangeToken) IsExpired() bool {
	return t.ExpiresAt.Before(time.Now())
}

// IsUsed はトークンが使用済みかを返す。
func (t *EmailChangeToken) IsUsed() bool {
	return t.UsedAt != nil
}
