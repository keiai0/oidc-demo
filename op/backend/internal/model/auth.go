package model

import "github.com/google/uuid"

type LoginInput struct {
	TenantCode string
	LoginID    string
	Password   string
	IPAddress  string
	UserAgent  string
}

type LoginOutput struct {
	SessionID uuid.UUID
	User      *User
}
