package model

import (
	"time"

	"github.com/google/uuid"
)

// MfaConfig はユーザーの MFA 設定を表す。
// 認証方式ごとに 1 レコード（type でユニーク）。
type MfaConfig struct {
	ID         uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID     uuid.UUID   `gorm:"type:uuid;not null"`
	Type       string      `gorm:"type:varchar(31);not null"`
	Enabled    bool        `gorm:"not null;default:false"`
	VerifiedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time

	TotpConfig *TotpConfig `gorm:"foreignKey:MfaConfigID"`
}

func (MfaConfig) TableName() string { return "mfa_configs" }
