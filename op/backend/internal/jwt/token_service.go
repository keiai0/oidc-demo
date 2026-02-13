package jwt

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

type TokenService struct {
	keySvc KeyProvider
}

func NewTokenService(keySvc KeyProvider) *TokenService {
	return &TokenService{keySvc: keySvc}
}

func (s *TokenService) SignIDToken(ctx context.Context, claims *model.IDTokenClaims, lifetime time.Duration) (string, string, error) {
	kid, privKey, err := s.keySvc.GetActiveSigningKey(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to get signing key: %w", err)
	}

	now := time.Now()
	jti := uuid.New().String()

	builder := jwt.NewBuilder().
		Issuer(claims.Issuer).
		Subject(claims.Subject).
		Audience([]string{claims.Audience}).
		IssuedAt(now).
		Expiration(now.Add(lifetime)).
		JwtID(jti).
		Claim("auth_time", claims.AuthTime.Unix())

	if claims.Nonce != nil {
		builder = builder.Claim("nonce", *claims.Nonce)
	}
	if claims.ATHash != "" {
		builder = builder.Claim("at_hash", claims.ATHash)
	}

	token, err := builder.Build()
	if err != nil {
		return "", "", fmt.Errorf("failed to build ID token: %w", err)
	}

	hdrs := jws.NewHeaders()
	_ = hdrs.Set(jws.KeyIDKey, kid)

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256(), privKey, jws.WithProtectedHeaders(hdrs)))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign ID token: %w", err)
	}

	return jti, string(signed), nil
}

func (s *TokenService) SignAccessToken(ctx context.Context, claims *model.AccessTokenClaims, lifetime time.Duration) (string, string, error) {
	kid, privKey, err := s.keySvc.GetActiveSigningKey(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to get signing key: %w", err)
	}

	now := time.Now()
	jti := uuid.New().String()

	token, err := jwt.NewBuilder().
		Issuer(claims.Issuer).
		Subject(claims.Subject).
		Audience([]string{claims.Audience}).
		IssuedAt(now).
		Expiration(now.Add(lifetime)).
		JwtID(jti).
		Claim("scope", claims.Scope).
		Claim("sid", claims.SessionID).
		Build()
	if err != nil {
		return "", "", fmt.Errorf("failed to build access token: %w", err)
	}

	hdrs := jws.NewHeaders()
	_ = hdrs.Set(jws.KeyIDKey, kid)

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256(), privKey, jws.WithProtectedHeaders(hdrs)))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return jti, string(signed), nil
}

func (s *TokenService) GenerateRefreshToken() (string, string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)
	hash := SHA256Hex(token)
	return token, hash, nil
}

func (s *TokenService) ValidateAccessToken(ctx context.Context, tokenString string) (*model.AccessTokenResult, error) {
	jwkSet, err := s.keySvc.GetJWKSet(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWK set: %w", err)
	}

	token, err := jwt.Parse([]byte(tokenString), jwt.WithKeySet(jwkSet))
	if err != nil {
		return nil, fmt.Errorf("failed to parse/verify access token: %w", err)
	}

	jti, _ := token.JwtID()
	sub, _ := token.Subject()

	var scope string
	_ = token.Get("scope", &scope)

	var sid string
	_ = token.Get("sid", &sid)

	aud, _ := token.Audience()
	clientID := ""
	if len(aud) > 0 {
		clientID = aud[0]
	}

	subUUID, err := uuid.Parse(sub)
	if err != nil {
		return nil, fmt.Errorf("invalid subject in access token: %w", err)
	}

	sessionUUID, err := uuid.Parse(sid)
	if err != nil {
		return nil, fmt.Errorf("invalid session id in access token: %w", err)
	}

	return &model.AccessTokenResult{
		JTI:       jti,
		Subject:   subUUID,
		ClientID:  clientID,
		Scope:     scope,
		SessionID: sessionUUID,
	}, nil
}

// ComputeATHash は at_hash を計算する (OIDC Core 1.0 Section 3.1.3.6)
// RS256 の場合、SHA-256 の左半分を base64url エンコード
func ComputeATHash(accessToken string) string {
	hash := sha256.Sum256([]byte(accessToken))
	leftHalf := hash[:len(hash)/2]
	return base64.RawURLEncoding.EncodeToString(leftHalf)
}

// SHA256Hex は文字列の SHA-256 ハッシュを hex エンコードで返す。
func SHA256Hex(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}
