package model

import (
	"time"

	"github.com/google/uuid"
)

// InitialAccessToken は Dynamic Client Registration (RFC 7591) で
// クライアント登録を許可するために管理者が発行するトークン。
type InitialAccessToken struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TokenHash        string     `gorm:"type:varchar(512);not null"`
	TenantID         uuid.UUID  `gorm:"type:uuid;not null"`
	MaxRegistrations int        `gorm:"not null;default:0"`
	UsedCount        int        `gorm:"not null;default:0"`
	ExpiresAt        time.Time  `gorm:"not null"`
	RevokedAt        *time.Time `gorm:"type:timestamptz"`
	CreatedAt        time.Time
	UpdatedAt        time.Time

	Tenant Tenant `gorm:"foreignKey:TenantID"`
}

func (InitialAccessToken) TableName() string { return "initial_access_tokens" }

// IsExpired はトークンが有効期限切れかどうかを返す。
func (t *InitialAccessToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsRevoked はトークンが無効化されているかどうかを返す。
func (t *InitialAccessToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

// IsExhausted は使用回数が上限に達しているかどうかを返す。
// MaxRegistrations が 0 の場合は無制限。
func (t *InitialAccessToken) IsExhausted() bool {
	return t.MaxRegistrations > 0 && t.UsedCount >= t.MaxRegistrations
}

// IsValid はトークンが有効かどうかを総合判定する。
func (t *InitialAccessToken) IsValid() bool {
	return !t.IsExpired() && !t.IsRevoked() && !t.IsExhausted()
}
