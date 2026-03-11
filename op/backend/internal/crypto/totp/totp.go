// Package totp は RFC 6238 (TOTP) / RFC 4226 (HOTP) に準拠した
// Time-based One-Time Password の生成・検証を提供する。
package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

const (
	// DefaultDigits はTOTPコードの桁数（RFC 6238 推奨値）。
	DefaultDigits = 6
	// DefaultPeriod はTOTPステップ間隔（秒）。
	DefaultPeriod = 30
	// SecretSize はシークレットのバイト数（RFC 4226 推奨: 160 bit = 20 bytes）。
	SecretSize = 20
	// ValidationWindow はコード検証時に許容するステップ幅（前後1ステップ）。
	ValidationWindow = 1
)

// GenerateSecret は暗号学的に安全なランダムシークレットを生成する。
// RFC 4226 Section 4: シークレットは最低 128 bit、推奨 160 bit。
func GenerateSecret() ([]byte, error) {
	secret := make([]byte, SecretSize)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}
	return secret, nil
}

// GenerateCode は指定されたステップ番号に対する HOTP コードを生成する。
// RFC 4226 Section 5.3: HMAC-SHA1 → Dynamic Truncation → mod 10^digits。
func GenerateCode(secret []byte, step int64, digits int) string {
	// Step 1: カウンタ値を 8 バイトのビッグエンディアンに変換
	msg := make([]byte, 8)
	binary.BigEndian.PutUint64(msg, uint64(step))

	// Step 2: HMAC-SHA1 計算
	mac := hmac.New(sha1.New, secret)
	mac.Write(msg)
	hash := mac.Sum(nil)

	// Step 3: Dynamic Truncation (RFC 4226 Section 5.3)
	offset := hash[len(hash)-1] & 0x0f
	binCode := int64(hash[offset]&0x7f)<<24 |
		int64(hash[offset+1]&0xff)<<16 |
		int64(hash[offset+2]&0xff)<<8 |
		int64(hash[offset+3]&0xff)

	// Step 4: mod 10^digits
	code := binCode % int64(math.Pow10(digits))

	return fmt.Sprintf("%0*d", digits, code)
}

// Validate は TOTP コードを検証する。
// ±ValidationWindow ステップの範囲でコードを照合し、
// リプレイ攻撃防止のため lastUsedStep 以下のステップは拒否する（RFC 6238 Section 5.2）。
//
// 戻り値:
//   - step: 一致したステップ番号（valid=true の場合のみ有効）
//   - valid: コードが有効かどうか
func Validate(secret []byte, code string, now time.Time, period, digits int, lastUsedStep *int64) (step int64, valid bool) {
	currentStep := now.Unix() / int64(period)

	for i := -ValidationWindow; i <= ValidationWindow; i++ {
		testStep := currentStep + int64(i)
		if testStep < 0 {
			continue
		}

		// リプレイ攻撃防止: 既に使用されたステップ以下は拒否
		if lastUsedStep != nil && testStep <= *lastUsedStep {
			continue
		}

		expected := GenerateCode(secret, testStep, digits)
		if hmac.Equal([]byte(code), []byte(expected)) {
			return testStep, true
		}
	}

	return 0, false
}
