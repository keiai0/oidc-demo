package model

import "time"

// DPoPJTICache は DPoP proof JWT の JTI リプレイ防止キャッシュ。
type DPoPJTICache struct {
	JTI       string    `gorm:"type:varchar(255);primaryKey"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
}

func (DPoPJTICache) TableName() string { return "dpop_jti_cache" }
