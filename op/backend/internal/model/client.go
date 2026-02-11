package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// StringSlice は JSONB カラムを []string としてマッピングするカスタム型
type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("StringSlice.Scan: type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, s)
}

type Client struct {
	ID                      uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID                uuid.UUID   `gorm:"type:uuid;not null;index"`
	ClientID                string      `gorm:"type:varchar(255);uniqueIndex;not null"`
	ClientSecretHash        string      `gorm:"type:varchar(512);not null"`
	Name                    string      `gorm:"type:varchar(255);not null"`
	GrantTypes              StringSlice `gorm:"type:jsonb;not null"`
	ResponseTypes           StringSlice `gorm:"type:jsonb;not null"`
	TokenEndpointAuthMethod string      `gorm:"type:varchar(63);not null;default:'client_secret_basic'"`
	RequirePKCE             bool        `gorm:"not null;default:true"`
	FrontchannelLogoutURI   *string     `gorm:"type:varchar(2048)"`
	BackchannelLogoutURI    *string     `gorm:"type:varchar(2048)"`
	Status                  string      `gorm:"type:varchar(31);not null;default:'active'"`
	CreatedAt               time.Time
	UpdatedAt               time.Time

	Tenant                Tenant                  `gorm:"foreignKey:TenantID"`
	RedirectURIs          []RedirectURI           `gorm:"foreignKey:ClientDBID"`
	PostLogoutRedirectURIs []PostLogoutRedirectURI `gorm:"foreignKey:ClientDBID"`
}

func (Client) TableName() string { return "clients" }

// HasGrantType はクライアントが指定された grant_type を許可しているか確認する
func (c *Client) HasGrantType(grantType string) bool {
	for _, gt := range c.GrantTypes {
		if gt == grantType {
			return true
		}
	}
	return false
}
