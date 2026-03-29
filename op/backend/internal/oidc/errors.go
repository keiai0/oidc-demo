package oidc

import "errors"

var (
	ErrInvalidClient        = errors.New("invalid_client")
	ErrInvalidGrant         = errors.New("invalid_grant")
	ErrUnsupportedGrantType = errors.New("unsupported_grant_type")
	ErrInvalidScope         = errors.New("invalid_scope")
	ErrInvalidDPoPProof     = errors.New("invalid_dpop_proof")
	ErrUseDPoPScheme        = errors.New("use_dpop_nonce")

	// Token Exchange (RFC 8693 Section 2.2.2)
	ErrInvalidTarget = errors.New("invalid_target")

	// Device Authorization Grant (RFC 8628 Section 3.5)
	ErrAuthorizationPending = errors.New("authorization_pending")
	ErrSlowDown             = errors.New("slow_down")
	ErrExpiredToken         = errors.New("expired_token")
	ErrAccessDenied         = errors.New("access_denied")
)
