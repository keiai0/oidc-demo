package audit

// 監査ログのイベント種別定数。
const (
	EventLoginSuccess      = "login_success"
	EventLoginFailure      = "login_failure"
	EventLoginLocked       = "login_locked"
	EventMFASuccess        = "mfa_verify_success"
	EventMFAFailure        = "mfa_verify_failure"
	EventTokenIssued       = "token_issued"
	EventTokenRevoked      = "token_revoked"
	EventSessionCreated    = "session_created"
	EventSessionDestroyed  = "session_destroyed"
	EventKeyRotated        = "key_rotated"
	EventAdminLogin        = "admin_login"
	EventAdminLoginFailure = "admin_login_failure"

	// Device Authorization Grant (RFC 8628)
	EventDeviceAuthorization = "device_authorization"
	EventDeviceApproved      = "device_approved"
	EventDeviceDenied        = "device_denied"

	// Federation (外部 IdP 連携)
	EventFederationLogin     = "federation_login"
	EventFederationProvision = "federation_provision"
)
