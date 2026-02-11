package crypto

import (
	"crypto/sha256"
	"encoding/base64"
)

// VerifyCodeChallenge は PKCE S256 を検証する。
// RFC 7636 Section 4.6: code_challenge = BASE64URL(SHA256(code_verifier))
func VerifyCodeChallenge(codeVerifier, codeChallenge string) bool {
	hash := sha256.Sum256([]byte(codeVerifier))
	computed := base64.RawURLEncoding.EncodeToString(hash[:])
	return computed == codeChallenge
}
