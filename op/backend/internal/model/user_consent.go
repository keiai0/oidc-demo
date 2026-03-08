package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type UserConsent struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null"`
	ClientID  uuid.UUID  `gorm:"type:uuid;not null"`
	Scopes    string     `gorm:"type:varchar(1024);not null"`
	GrantedAt time.Time  `gorm:"not null;default:now()"`
	RevokedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time

	User   User   `gorm:"foreignKey:UserID"`
	Client Client `gorm:"foreignKey:ClientID"`
}

func (UserConsent) TableName() string { return "user_consents" }

// CoversScopes は同意済みスコープが要求スコープを全てカバーしているかを返す。
func (c *UserConsent) CoversScopes(requestedScopes []string) bool {
	if c.RevokedAt != nil {
		return false
	}
	granted := make(map[string]bool)
	for _, s := range strings.Split(c.Scopes, " ") {
		granted[s] = true
	}
	for _, s := range requestedScopes {
		if !granted[s] {
			return false
		}
	}
	return true
}
