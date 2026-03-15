package model

import "github.com/google/uuid"

type LoginInput struct {
	TenantCode   string
	LoginID      string
	Password     string
	IPAddress    string
	UserAgent    string
	OldSessionID *uuid.UUID // セッション固定攻撃対策: ログイン前の既存セッションを失効させる
}

type LoginOutput struct {
	SessionID        uuid.UUID
	User             *User
	MFARequired      bool
	MFASetupRequired bool
	MFAMethods        []string
	PasskeyRegistered bool
}
