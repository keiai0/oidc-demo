package model

// AuthorizationDetail は RFC 9396 Section 2 で定義される authorization_details の各要素。
// JSON のパース・シリアライズに使用する DTO。
type AuthorizationDetail struct {
	Type       string                 `json:"type"`
	Locations  []string               `json:"locations,omitempty"`
	Actions    []string               `json:"actions,omitempty"`
	DataTypes  []string               `json:"datatypes,omitempty"`
	Identifier string                 `json:"identifier,omitempty"`
	Privileges []string               `json:"privileges,omitempty"`
	Extra      map[string]interface{} `json:"-"`
}
