package model

// IDTokenHintResult は id_token_hint の検証結果。
// 署名検証のみ行い、exp は検証しない（RP-Initiated Logout 1.0 では期限切れの ID トークンが渡される場合がある）。
type IDTokenHintResult struct {
	Issuer    string
	Subject   string
	Audience  string
	SessionID string // sid クレーム（存在する場合）
}
