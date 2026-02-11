package model

import "github.com/google/uuid"

type AccessTokenClaims struct {
	Issuer    string
	Subject   string
	Audience  string
	Scope     string
	SessionID string
}

type AccessTokenResult struct {
	JTI       string
	Subject   uuid.UUID
	ClientID  string
	Scope     string
	SessionID uuid.UUID
}
