package model

import (
	"time"

	"github.com/google/uuid"
)

// AdminUser は管理コンソールの管理者を表す。
// OIDC ユーザー（users テーブル）とは設計上分離 — OP ユーザーモデルにはロール/権限フィールドがない。
type AdminUser struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	LoginID      string     `gorm:"type:varchar(255);not null;uniqueIndex"`
	PasswordHash string     `gorm:"type:text;not null"`
	Name         string     `gorm:"type:varchar(255);not null"`
	Status       string     `gorm:"type:varchar(31);not null;default:'active'"`
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (AdminUser) TableName() string { return "admin_users" }
