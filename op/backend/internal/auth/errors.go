package auth

import "errors"

var (
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrAccountLocked           = errors.New("account locked")
	ErrSessionNotFound         = errors.New("session not found")
	ErrSessionExpired          = errors.New("session expired or revoked")
	ErrEmailChangeTokenInvalid = errors.New("email change token is invalid or expired")
	ErrBackupCodeInvalid       = errors.New("backup code is invalid or already used")
)
