package model

import (
	"time"

	"github.com/google/uuid"
)

// TokenExchangePolicy は Token Exchange (RFC 8693) のクライアントごとのポリシー。
type TokenExchangePolicy struct {
	ID                         uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ClientID                   uuid.UUID   `gorm:"type:uuid;uniqueIndex;not null" json:"client_id"`
	AllowedSubjectTokenTypes   StringSlice `gorm:"type:jsonb;not null" json:"allowed_subject_token_types"`
	AllowedRequestedTokenTypes StringSlice `gorm:"type:jsonb;not null" json:"allowed_requested_token_types"`
	AllowedAudiences           StringSlice `gorm:"type:jsonb;not null" json:"allowed_audiences"`
	AllowImpersonation         bool        `gorm:"not null;default:false" json:"allow_impersonation"`
	AllowDelegation            bool        `gorm:"not null;default:true" json:"allow_delegation"`
	CreatedAt                  time.Time   `json:"created_at"`
	UpdatedAt                  time.Time   `json:"updated_at"`

	Client Client `gorm:"foreignKey:ClientID" json:"-"`
}

// HasSubjectTokenType は指定された subject_token_type が許可されているか返す。
func (p *TokenExchangePolicy) HasSubjectTokenType(tokenType string) bool {
	for _, t := range p.AllowedSubjectTokenTypes {
		if t == tokenType {
			return true
		}
	}
	return false
}

// HasRequestedTokenType は指定された requested_token_type が許可されているか返す。
func (p *TokenExchangePolicy) HasRequestedTokenType(tokenType string) bool {
	for _, t := range p.AllowedRequestedTokenTypes {
		if t == tokenType {
			return true
		}
	}
	return false
}

// HasAudience は指定された audience が許可されているか返す。
// AllowedAudiences が空の場合は制限なし（常に true）。
func (p *TokenExchangePolicy) HasAudience(audience string) bool {
	if len(p.AllowedAudiences) == 0 {
		return true
	}
	for _, a := range p.AllowedAudiences {
		if a == audience {
			return true
		}
	}
	return false
}
