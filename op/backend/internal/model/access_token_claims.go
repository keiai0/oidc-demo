package model

import "github.com/google/uuid"

// TokenConfirmation は DPoP の cnf クレーム (RFC 9449 Section 6.1)。
type TokenConfirmation struct {
	JKT string `json:"jkt"`
}

// ActClaim は Token Exchange (RFC 8693 Section 4.1) の act クレーム。
// 委任チェーンをネスト構造で表現する。
type ActClaim struct {
	Sub      string   `json:"sub"`
	ClientID string   `json:"client_id,omitempty"`
	Act      *ActClaim `json:"act,omitempty"`
}

type AccessTokenClaims struct {
	Issuer               string
	Subject              string
	Audience             string
	Scope                string
	SessionID            string
	Confirmation         *TokenConfirmation  // DPoP: cnf.jkt
	Act                  *ActClaim           // RFC 8693: delegation chain
	AuthorizationDetails []AuthorizationDetail // RFC 9396: rich authorization requests
}

// AccessTokenResult はアクセストークン検証結果。
// Subject は pairwise sub (base64url hash) や client_credentials (client_id) の場合もあるため string。
type AccessTokenResult struct {
	JTI                  string
	Subject              string
	ClientID             string
	Scope                string
	SessionID            *uuid.UUID          // nil for client_credentials grant
	DPoPJKT              *string             // DPoP: cnf.jkt from token
	Act                  *ActClaim           // RFC 8693: delegation chain (nil if not a delegated token)
	AuthorizationDetails []AuthorizationDetail // RFC 9396: rich authorization requests (nil if not present)
}
