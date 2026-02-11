package model

import (
	"time"

	"github.com/google/uuid"
)

type SignKey struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	KID           string     `gorm:"column:kid;type:varchar(255);uniqueIndex;not null"`
	Algorithm     string     `gorm:"type:varchar(31);not null;default:'RS256'"`
	PublicKey     string     `gorm:"type:text;not null"`
	PrivateKeyRef string     `gorm:"type:text;not null"`
	Active        bool       `gorm:"not null;default:true"`
	CreatedAt     time.Time
	RotatedAt     *time.Time
}

func (SignKey) TableName() string { return "sign_keys" }
