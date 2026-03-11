package model

import "github.com/google/uuid"

// TotpConfig は TOTP 設定を表す（mfa_configs の子テーブル）。
type TotpConfig struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	MfaConfigID        uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	SecretKeyEncrypted string    `gorm:"type:text;not null"`
	Algorithm          string    `gorm:"type:varchar(15);not null;default:'SHA1'"`
	Digits             int16     `gorm:"not null;default:6"`
	Period             int16     `gorm:"not null;default:30"`
	LastUsedStep       *int64
}

func (TotpConfig) TableName() string { return "totp_configs" }
