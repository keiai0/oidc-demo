package oidc

import (
	"encoding/json"
	"fmt"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// ParseClaimsRequest は claims リクエストパラメータの JSON をパースする。
// 仕様参照: OIDC Core 1.0 Section 5.5
func ParseClaimsRequest(raw string) (*model.ClaimsRequest, error) {
	if raw == "" {
		return nil, nil
	}

	var cr model.ClaimsRequest
	if err := json.Unmarshal([]byte(raw), &cr); err != nil {
		return nil, fmt.Errorf("invalid claims parameter: %w", err)
	}
	return &cr, nil
}

// サポートするクレーム名の一覧
var supportedUserClaims = map[string]bool{
	"name":           true,
	"email":          true,
	"email_verified": true,
	"updated_at":     true,
}

var supportedSessionClaims = map[string]bool{
	"auth_time": true,
	"acr":       true,
	"amr":       true,
}

// ResolveIDTokenClaims は claims パラメータの id_token ターゲットからクレームを解決する。
// セッション由来のクレーム (auth_time, acr, amr) とユーザー由来のクレームを返す。
func ResolveIDTokenClaims(cr *model.ClaimsRequest, user *model.User, session *model.Session) map[string]interface{} {
	if cr == nil || cr.IDToken == nil {
		return nil
	}

	result := make(map[string]interface{})

	for claimName := range cr.IDToken {
		if val, ok := resolveUserClaim(claimName, user); ok {
			result[claimName] = val
		}
		if val, ok := resolveSessionClaim(claimName, session); ok {
			result[claimName] = val
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// ResolveUserinfoClaims は claims パラメータの userinfo ターゲットからクレームを解決する。
func ResolveUserinfoClaims(cr *model.ClaimsRequest, user *model.User) map[string]interface{} {
	if cr == nil || cr.Userinfo == nil {
		return nil
	}

	result := make(map[string]interface{})

	for claimName := range cr.Userinfo {
		if val, ok := resolveUserClaim(claimName, user); ok {
			result[claimName] = val
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func resolveUserClaim(name string, user *model.User) (interface{}, bool) {
	if !supportedUserClaims[name] || user == nil {
		return nil, false
	}

	switch name {
	case "name":
		if user.Name != nil {
			return *user.Name, true
		}
		return nil, false
	case "email":
		return user.Email, true
	case "email_verified":
		return user.EmailVerified, true
	case "updated_at":
		return user.UpdatedAt.Unix(), true
	}
	return nil, false
}

func resolveSessionClaim(name string, session *model.Session) (interface{}, bool) {
	if !supportedSessionClaims[name] || session == nil {
		return nil, false
	}

	switch name {
	case "auth_time":
		return session.AuthTime.Unix(), true
	case "acr":
		if session.ACR != "" {
			return session.ACR, true
		}
		return nil, false
	case "amr":
		if len(session.AMR) > 0 {
			return []string(session.AMR), true
		}
		return nil, false
	}
	return nil, false
}
