package model

import "time"

type IDTokenClaims struct {
	Issuer      string
	Subject     string
	Audience    string
	Nonce       *string
	AuthTime    time.Time
	ATHash      string
	ACR         string
	AMR         []string
	SessionID   string
	ExtraClaims map[string]interface{} // claims リクエストパラメータで要求された追加クレーム
}
