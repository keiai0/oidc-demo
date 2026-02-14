package oidc

import "errors"

var (
	ErrInvalidClient        = errors.New("invalid_client")
	ErrInvalidGrant         = errors.New("invalid_grant")
	ErrUnsupportedGrantType = errors.New("unsupported_grant_type")
)
