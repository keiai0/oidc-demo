package model

// LogoutTokenClaims は Back-Channel Logout で送信する logout_token の入力。
// 仕様参照: OIDC Back-Channel Logout 1.0 Section 2.4
type LogoutTokenClaims struct {
	Issuer    string
	Subject   string
	Audience  string
	SessionID string
}
