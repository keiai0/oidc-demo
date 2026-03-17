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

// AccessTokenResult はアクセストークン検証結果。
// Subject は pairwise sub (base64url hash) や client_credentials (client_id) の場合もあるため string。
type AccessTokenResult struct {
	JTI       string
	Subject   string
	ClientID  string
	Scope     string
	SessionID *uuid.UUID // nil for client_credentials grant
	DPoPJKT   *string    // DPoP: cnf.jkt from token
}
