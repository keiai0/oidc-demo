package model

import (
	"time"

	"github.com/google/uuid"
)

type PostLogoutRedirectURI struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ClientDBID uuid.UUID `gorm:"type:uuid;not null;index;column:client_id"`
	URI        string    `gorm:"type:varchar(2048);not null"`
	CreatedAt  time.Time
}

func (PostLogoutRedirectURI) TableName() string { return "post_logout_redirect_uris" }
