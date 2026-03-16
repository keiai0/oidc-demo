package model

import (
	"time"

	"github.com/google/uuid"
)

// 署名鍵のライフサイクル状態定数。
const (
	SignKeyStatusActive  = "active"  // 署名・検証に使用。JWKS に含む。
	SignKeyStatusPassive = "passive" // 検証のみ（猶予期間）。JWKS に含む。
	SignKeyStatusExpired = "expired" // 削除対象。JWKS に含まない。
)

type SignKey struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	KID           string     `gorm:"column:kid;type:varchar(255);uniqueIndex;not null"`
	Algorithm     string     `gorm:"type:varchar(31);not null;default:'RS256'"`
	PublicKey     string     `gorm:"type:text;not null"`
	PrivateKeyRef string     `gorm:"type:text;not null"`
	Status        string     `gorm:"type:varchar(31);not null;default:'active'"`
	CreatedAt     time.Time
	RotatedAt     *time.Time
}

func (SignKey) TableName() string { return "sign_keys" }
