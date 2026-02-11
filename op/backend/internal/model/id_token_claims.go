package model

import "time"

type IDTokenClaims struct {
	Issuer   string
	Subject  string
	Audience string
	Nonce    *string
	AuthTime time.Time
	ATHash   string
}
