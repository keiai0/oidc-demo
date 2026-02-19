package jwt

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"

	infra_crypto "github.com/isurugi-k/oidc-demo/op/backend/internal/crypto"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

type KeyService struct {
	signKeyRepo SignKeyRepository
	encKey      []byte
}

func NewKeyService(repo SignKeyRepository, encKeyHex string) (*KeyService, error) {
	encKey, err := hex.DecodeString(encKeyHex)
	if err != nil || len(encKey) != 32 {
		return nil, fmt.Errorf("OP_KEY_ENCRYPTION_KEY must be 64 hex characters (32 bytes)")
	}
	return &KeyService{signKeyRepo: repo, encKey: encKey}, nil
}

func (s *KeyService) EnsureSigningKey(ctx context.Context) error {
	existing, err := s.signKeyRepo.FindActive(ctx)
	if err != nil {
		return fmt.Errorf("failed to check existing key: %w", err)
	}
	if existing != nil {
		return nil
	}

	_, err = s.generateAndSaveKey(ctx)
	return err
}

// RotateKey は新しい署名鍵を生成・保存し、既存の有効な鍵を全て無効化する。
func (s *KeyService) RotateKey(ctx context.Context) (*model.SignKey, error) {
	// 既存のアクティブ鍵を全て無効化
	if err := s.signKeyRepo.DeactivateAllActive(ctx); err != nil {
		return nil, fmt.Errorf("failed to deactivate existing keys: %w", err)
	}

	// 新しい鍵を生成・保存
	newKey, err := s.generateAndSaveKey(ctx)
	if err != nil {
		return nil, err
	}

	return newKey, nil
}

// generateAndSaveKey は RSA 2048 ビット鍵ペアを生成し、秘密鍵を暗号化して永続化する。
func (s *KeyService) generateAndSaveKey(ctx context.Context) (*model.SignKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// 秘密鍵を PEM エンコード → AES-GCM で暗号化
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	encryptedPriv, err := infra_crypto.Encrypt(privPEM, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt private key: %w", err)
	}

	// 公開鍵を PEM エンコード
	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}
	pubPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}))

	// kid 生成: {date}-{random8hex}
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("failed to generate kid: %w", err)
	}
	kid := fmt.Sprintf("%s-%s", time.Now().Format("2006-01-02"), hex.EncodeToString(randomBytes))

	signKey := &model.SignKey{
		KID:           kid,
		Algorithm:     "RS256",
		PublicKey:     pubPEM,
		PrivateKeyRef: encryptedPriv,
		Active:        true,
	}

	if err := s.signKeyRepo.Create(ctx, signKey); err != nil {
		return nil, fmt.Errorf("failed to save signing key: %w", err)
	}

	return signKey, nil
}

func (s *KeyService) GetActiveSigningKey(ctx context.Context) (string, crypto.PrivateKey, error) {
	key, err := s.signKeyRepo.FindActive(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to find active key: %w", err)
	}
	if key == nil {
		return "", nil, fmt.Errorf("no active signing key found")
	}

	privPEM, err := infra_crypto.Decrypt(key.PrivateKeyRef, s.encKey)
	if err != nil {
		return "", nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	block, _ := pem.Decode(privPEM)
	if block == nil {
		return "", nil, fmt.Errorf("failed to decode PEM block")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return key.KID, privateKey, nil
}

func (s *KeyService) GetJWKSet(ctx context.Context) (jwk.Set, error) {
	keys, err := s.signKeyRepo.FindAllActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find active keys: %w", err)
	}

	set := jwk.NewSet()
	for _, k := range keys {
		block, _ := pem.Decode([]byte(k.PublicKey))
		if block == nil {
			continue
		}

		pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			continue
		}

		jwkKey, err := jwk.Import(pubKey)
		if err != nil {
			continue
		}

		if err := jwkKey.Set(jwk.KeyIDKey, k.KID); err != nil {
			continue
		}
		if err := jwkKey.Set(jwk.AlgorithmKey, jwa.RS256()); err != nil {
			continue
		}
		if err := jwkKey.Set(jwk.KeyUsageKey, "sig"); err != nil {
			continue
		}

		if err := set.AddKey(jwkKey); err != nil {
			continue
		}
	}

	return set, nil
}
