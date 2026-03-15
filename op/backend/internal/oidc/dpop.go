package oidc

import (
	"context"
	"crypto"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

// DPoPResult は DPoP proof の検証結果。
type DPoPResult struct {
	JKT string // JWK Thumbprint (base64url, RFC 7638)
}

// dpopMaxClockSkew は DPoP proof の iat に許容するクロックスキュー。
const dpopMaxClockSkew = 5 * time.Minute

// supportedDPoPAlgorithms は DPoP proof で許可する署名アルゴリズム。
var supportedDPoPAlgorithms = map[jwa.SignatureAlgorithm]bool{
	jwa.RS256(): true,
	jwa.ES256(): true,
}

// VerifyDPoPProof は DPoP proof JWT を検証し、JWK Thumbprint を返す。
// 仕様参照: RFC 9449 Section 4.3
func VerifyDPoPProof(ctx context.Context, proofStr string, httpMethod string, httpURL string, jtiStore DPoPJTIStore) (*DPoPResult, error) {
	// 1. JWS を署名検証なしでまずパースし、ヘッダーの jwk を取得
	msg, err := jws.Parse([]byte(proofStr))
	if err != nil {
		return nil, fmt.Errorf("failed to parse DPoP proof: %w", ErrInvalidDPoPProof)
	}

	sigs := msg.Signatures()
	if len(sigs) != 1 {
		return nil, fmt.Errorf("DPoP proof must have exactly one signature: %w", ErrInvalidDPoPProof)
	}

	sig := sigs[0]
	hdrs := sig.ProtectedHeaders()

	// 2. typ ヘッダー: MUST be "dpop+jwt" (RFC 9449 Section 4.3)
	var typ string
	if err := hdrs.Get("typ", &typ); err != nil || typ != "dpop+jwt" {
		return nil, fmt.Errorf("DPoP proof typ must be dpop+jwt: %w", ErrInvalidDPoPProof)
	}

	// 3. alg ヘッダー: サポートするアルゴリズムか確認
	alg, ok := hdrs.Algorithm()
	if !ok {
		return nil, fmt.Errorf("DPoP proof missing alg header: %w", ErrInvalidDPoPProof)
	}
	if !supportedDPoPAlgorithms[alg] {
		return nil, fmt.Errorf("unsupported DPoP signing algorithm: %s: %w", alg.String(), ErrInvalidDPoPProof)
	}

	// 4. jwk ヘッダーから公開鍵を取得（jwx v3 では jwk.Key 型で格納されている）
	var pubKey jwk.Key
	if err := hdrs.Get(jws.JWKKey, &pubKey); err != nil || pubKey == nil {
		return nil, fmt.Errorf("DPoP proof missing jwk header: %w", ErrInvalidDPoPProof)
	}

	// 5. 公開鍵で署名検証
	token, err := jwt.Parse([]byte(proofStr), jwt.WithKey(alg, pubKey))
	if err != nil {
		return nil, fmt.Errorf("DPoP proof signature verification failed: %w", ErrInvalidDPoPProof)
	}

	// 6. htm (HTTP メソッド) 検証 (MUST: RFC 9449 Section 4.3)
	var htm string
	if err := token.Get("htm", &htm); err != nil || htm != httpMethod {
		return nil, fmt.Errorf("DPoP htm mismatch: %w", ErrInvalidDPoPProof)
	}

	// 7. htu (HTTP URL) 検証 (MUST: RFC 9449 Section 4.3)
	var htu string
	if err := token.Get("htu", &htu); err != nil || htu != httpURL {
		return nil, fmt.Errorf("DPoP htu mismatch: %w", ErrInvalidDPoPProof)
	}

	// 8. iat 検証: ±5分以内
	iat, ok := token.IssuedAt()
	if !ok {
		return nil, fmt.Errorf("DPoP proof missing iat: %w", ErrInvalidDPoPProof)
	}
	now := time.Now()
	if now.Sub(iat) > dpopMaxClockSkew || iat.Sub(now) > dpopMaxClockSkew {
		return nil, fmt.Errorf("DPoP proof iat out of range: %w", ErrInvalidDPoPProof)
	}

	// 9. jti リプレイ防止 (SHOULD: RFC 9449 Section 4.3)
	jti, ok := token.JwtID()
	if !ok || jti == "" {
		return nil, fmt.Errorf("DPoP proof missing jti: %w", ErrInvalidDPoPProof)
	}

	exists, err := jtiStore.Exists(ctx, jti)
	if err != nil {
		return nil, fmt.Errorf("failed to check DPoP jti: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("DPoP proof jti replay detected: %w", ErrInvalidDPoPProof)
	}

	if err := jtiStore.Create(ctx, jti); err != nil {
		return nil, fmt.Errorf("failed to cache DPoP jti: %w", err)
	}

	// 10. JWK Thumbprint 計算 (RFC 7638)
	thumbprint, err := pubKey.Thumbprint(crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("failed to compute JWK thumbprint: %w", err)
	}
	jkt := base64.RawURLEncoding.EncodeToString(thumbprint)

	return &DPoPResult{JKT: jkt}, nil
}
