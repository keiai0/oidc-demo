package model

import (
	"time"

	"github.com/google/uuid"
)

// ClientRegistration は Dynamic Client Registration (RFC 7591/7592) で
// 登録されたクライアントの追加メタデータを保持する。
type ClientRegistration struct {
	ID                          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ClientID                    uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	RegistrationAccessTokenHash string     `gorm:"type:varchar(512);not null"`
	RegistrationClientURI       string     `gorm:"type:varchar(2048);not null"`
	SoftwareID                  *string    `gorm:"type:varchar(255)"`
	SoftwareVersion             *string    `gorm:"type:varchar(255)"`
	InitialAccessTokenID        *uuid.UUID `gorm:"type:uuid"`
	CreatedAt                   time.Time
	UpdatedAt                   time.Time

	Client             Client              `gorm:"foreignKey:ClientID"`
	InitialAccessToken *InitialAccessToken `gorm:"foreignKey:InitialAccessTokenID"`
}

func (ClientRegistration) TableName() string { return "client_registrations" }
