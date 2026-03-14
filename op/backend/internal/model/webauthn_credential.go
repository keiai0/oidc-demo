package model

import (
	"time"

	"github.com/google/uuid"
)

// WebAuthnCredential は WebAuthn（パスキー）のクレデンシャルを表す。
// 1ユーザーが複数デバイスを登録可能。
type WebAuthnCredential struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	MfaConfigID     uuid.UUID `gorm:"type:uuid;not null"`
	CredentialID    string    `gorm:"type:text;not null;uniqueIndex"`
	PublicKey       string    `gorm:"type:text;not null"`
	AttestationType string    `gorm:"type:text;not null;default:''"`
	AAGUID          string    `gorm:"column:aaguid;type:text;not null;default:''"`
	SignCount       uint32    `gorm:"not null;default:0"`
	BackupEligible  bool      `gorm:"not null;default:false"`
	BackupState     bool      `gorm:"not null;default:false"`
	Name            string    `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt       time.Time
}

func (WebAuthnCredential) TableName() string { return "webauthn_credentials" }
