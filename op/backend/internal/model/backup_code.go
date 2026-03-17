package model

import (
	"time"

	"github.com/google/uuid"
)

// BackupCode は MFA バックアップコードを表す。
// argon2id ハッシュで保存し、1回のみ使用可能。
// TOTP・WebAuthn など全ての MFA 手段を失った場合の最終手段として使用する。
type BackupCode struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	CodeHash  string     `gorm:"type:varchar(512);not null"`
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (BackupCode) TableName() string { return "backup_codes" }

// IsUsed はコードが使用済みかを返す。
func (c *BackupCode) IsUsed() bool {
	return c.UsedAt != nil
}
