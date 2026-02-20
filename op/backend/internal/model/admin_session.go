package model

import (
	"time"

	"github.com/google/uuid"
)

// AdminSession は管理コンソールのログインセッションを表す。
type AdminSession struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AdminUserID uuid.UUID  `gorm:"type:uuid;not null"`
	IPAddress   string     `gorm:"type:varchar(45);not null"`
	UserAgent   string     `gorm:"type:text;not null;default:''"`
	ExpiresAt   time.Time  `gorm:"not null"`
	RevokedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time

	AdminUser AdminUser `gorm:"foreignKey:AdminUserID"`
}

func (AdminSession) TableName() string { return "admin_sessions" }

// IsValid はセッションが失効済みでも期限切れでもない場合に true を返す。
func (s *AdminSession) IsValid() bool {
	return s.RevokedAt == nil && s.ExpiresAt.After(time.Now())
}
