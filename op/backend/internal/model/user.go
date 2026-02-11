package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID      uuid.UUID  `gorm:"type:uuid;not null"`
	LoginID       string     `gorm:"type:varchar(255);not null"`
	Email         string     `gorm:"type:varchar(255);not null;index"`
	EmailVerified bool       `gorm:"not null;default:false"`
	Name          *string    `gorm:"type:varchar(255)"`
	Status        string     `gorm:"type:varchar(31);not null;default:'active'"`
	LastLoginAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time

	Tenant      Tenant       `gorm:"foreignKey:TenantID"`
	Credentials []Credential `gorm:"foreignKey:UserID"`
}

func (User) TableName() string { return "users" }
