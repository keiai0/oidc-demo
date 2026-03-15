package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PushedAuthorizationRequest は PAR (RFC 9126) のリクエスト情報を保持する。
type PushedAuthorizationRequest struct {
	ID         uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RequestURI string          `gorm:"type:varchar(255);uniqueIndex;not null"`
	ClientID   uuid.UUID       `gorm:"type:uuid;not null"`
	Parameters json.RawMessage `gorm:"type:jsonb;not null"`
	ExpiresAt  time.Time       `gorm:"not null"`
	UsedAt     *time.Time

	Client Client `gorm:"foreignKey:ClientID"`
}

func (PushedAuthorizationRequest) TableName() string { return "pushed_authorization_requests" }

// IsExpired は PAR リクエストが有効期限切れかどうかを返す。
func (p *PushedAuthorizationRequest) IsExpired() bool {
	return p.ExpiresAt.Before(time.Now())
}

// IsUsed は PAR リクエストが使用済みかどうかを返す。
func (p *PushedAuthorizationRequest) IsUsed() bool {
	return p.UsedAt != nil
}
