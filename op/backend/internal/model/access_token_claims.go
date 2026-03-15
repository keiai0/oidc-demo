package model

import "github.com/google/uuid"

// TokenConfirmation は DPoP の cnf クレーム (RFC 9449 Section 6.1)。
type TokenConfirmation struct {
	JKT string `json:"jkt"`
}

type AccessTokenClaims struct {
	Issuer       string
	Subject      string
	Audience     string
	Scope        string
	SessionID    string
	Confirmation *TokenConfirmation // DPoP: cnf.jkt
}

type AccessTokenResult struct {
	JTI       string
	Subject   uuid.UUID
	ClientID  string
	Scope     string
	SessionID uuid.UUID
	DPoPJKT   *string // DPoP: cnf.jkt from token
}
