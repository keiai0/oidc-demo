package model

import (
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Code                 string    `gorm:"type:varchar(63);uniqueIndex;not null"`
	Name                 string    `gorm:"type:varchar(255);not null"`
	SessionLifetime      int       `gorm:"not null;default:3600"`
	AuthCodeLifetime     int       `gorm:"not null;default:60"`
	AccessTokenLifetime  int       `gorm:"not null;default:3600"`
	RefreshTokenLifetime int       `gorm:"not null;default:2592000"`
	IDTokenLifetime      int       `gorm:"not null;default:3600"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (Tenant) TableName() string { return "tenants" }
