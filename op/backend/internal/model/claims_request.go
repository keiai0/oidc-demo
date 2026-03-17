package model

// ClaimsRequest は OIDC claims リクエストパラメータの構造。
// 仕様参照: OIDC Core 1.0 Section 5.5
type ClaimsRequest struct {
	IDToken  map[string]*ClaimSpec `json:"id_token,omitempty"`
	Userinfo map[string]*ClaimSpec `json:"userinfo,omitempty"`
}

// ClaimSpec は個別クレームの要求仕様。
type ClaimSpec struct {
	Essential bool          `json:"essential,omitempty"`
	Value     interface{}   `json:"value,omitempty"`
	Values    []interface{} `json:"values,omitempty"`
}
