package model

import (
	"time"

	"github.com/google/uuid"
)

// ExternalIdPCredential は外部 IdP 経由の認証クレデンシャルを保持する。
// Credential テーブル（type=oidc_provider）の子テーブル。
type ExternalIdPCredential struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CredentialID    uuid.UUID `gorm:"type:uuid;not null"`
	ProviderID      uuid.UUID `gorm:"type:uuid;not null"`
	ProviderSubject string    `gorm:"type:varchar(255);not null"`
	CreatedAt       time.Time

	Credential         Credential         `gorm:"foreignKey:CredentialID"`
	FederationProvider FederationProvider `gorm:"foreignKey:ProviderID"`
}

func (ExternalIdPCredential) TableName() string { return "external_idp_credentials" }
