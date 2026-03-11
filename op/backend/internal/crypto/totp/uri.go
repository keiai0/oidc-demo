package totp

import (
	"encoding/base32"
	"fmt"
	"net/url"
)

// BuildOTPAuthURI は otpauth:// URI を生成する。
// この URI は QR コードにエンコードされ、認証アプリ（Google Authenticator 等）で読み取る。
//
// フォーマット: otpauth://totp/{issuer}:{account}?secret={base32}&issuer={issuer}&algorithm=SHA1&digits={digits}&period={period}
func BuildOTPAuthURI(issuer, account string, secret []byte, digits, period int) string {
	secretB32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)

	label := fmt.Sprintf("%s:%s", issuer, account)

	u := &url.URL{
		Scheme: "otpauth",
		Host:   "totp",
		Path:   label,
	}

	q := u.Query()
	q.Set("secret", secretB32)
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprintf("%d", digits))
	q.Set("period", fmt.Sprintf("%d", period))
	u.RawQuery = q.Encode()

	return u.String()
}
