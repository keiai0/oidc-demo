package auth

import (
	"context"
	"encoding/base32"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	qrcode "github.com/skip2/go-qrcode"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/crypto/totp"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

var (
	ErrMFAAlreadyConfigured = errors.New("MFA is already configured")
	ErrMFANotConfigured     = errors.New("MFA is not configured")
	ErrMFANotPending        = errors.New("session is not pending MFA")
	ErrInvalidTOTPCode      = errors.New("invalid TOTP code")
)

// TOTPSetupResult は TOTP セットアップの結果を格納する。
type TOTPSetupResult struct {
	Secret     string // base32 エンコード済みシークレット（手動入力用）
	QRCodeURI  string // otpauth:// URI
	QRCodePNG  []byte // QR コード PNG 画像
}

// MFATOTPService は TOTP MFA のビジネスロジックを提供する。
type MFATOTPService struct {
	mfaConfigFinder MfaConfigFinder
	mfaConfigStore  MfaConfigStore
	totpConfigStore TotpConfigStore
	sessionUpdater  SessionUpdater
	encrypt         EncryptFunc
	decrypt         DecryptFunc
	encryptionKey   []byte
	issuerName      string
}

// NewMFATOTPService は MFATOTPService を生成する。
func NewMFATOTPService(
	mfaConfigFinder MfaConfigFinder,
	mfaConfigStore MfaConfigStore,
	totpConfigStore TotpConfigStore,
	sessionUpdater SessionUpdater,
	encrypt EncryptFunc,
	decrypt DecryptFunc,
	encryptionKey []byte,
	issuerName string,
) *MFATOTPService {
	return &MFATOTPService{
		mfaConfigFinder: mfaConfigFinder,
		mfaConfigStore:  mfaConfigStore,
		totpConfigStore: totpConfigStore,
		sessionUpdater:  sessionUpdater,
		encrypt:         encrypt,
		decrypt:         decrypt,
		encryptionKey:   encryptionKey,
		issuerName:      issuerName,
	}
}

// Setup は TOTP セットアップを開始する。
// シークレットを生成し、暗号化して DB に保存し、QR コードを返す。
func (s *MFATOTPService) Setup(ctx context.Context, userID uuid.UUID, accountName string) (*TOTPSetupResult, error) {
	// 既存の TOTP 設定を確認
	existing, err := s.mfaConfigFinder.FindByUserIDAndType(ctx, userID, "totp")
	if err != nil {
		return nil, fmt.Errorf("failed to check existing MFA config: %w", err)
	}
	if existing != nil && existing.Enabled {
		return nil, ErrMFAAlreadyConfigured
	}

	// 既存の未検証設定があれば削除して再作成
	if existing != nil {
		if err := s.mfaConfigStore.Delete(ctx, existing.ID); err != nil {
			return nil, fmt.Errorf("failed to delete existing MFA config: %w", err)
		}
	}

	// シークレット生成
	secret, err := totp.GenerateSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	// AES-256-GCM で暗号化
	encrypted, err := s.encrypt(secret, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt TOTP secret: %w", err)
	}

	// MFA 設定レコード作成
	mfaConfig := &model.MfaConfig{
		UserID:  userID,
		Type:    "totp",
		Enabled: false,
	}
	if err := s.mfaConfigStore.Create(ctx, mfaConfig); err != nil {
		return nil, fmt.Errorf("failed to create MFA config: %w", err)
	}

	// TOTP 設定レコード作成
	totpConfig := &model.TotpConfig{
		MfaConfigID:        mfaConfig.ID,
		SecretKeyEncrypted: encrypted,
		Algorithm:          "SHA1",
		Digits:             totp.DefaultDigits,
		Period:             totp.DefaultPeriod,
	}
	if err := s.totpConfigStore.Create(ctx, totpConfig); err != nil {
		return nil, fmt.Errorf("failed to create TOTP config: %w", err)
	}

	// otpauth URI 生成
	uri := totp.BuildOTPAuthURI(s.issuerName, accountName, secret, totp.DefaultDigits, totp.DefaultPeriod)

	// QR コード生成
	png, err := qrcode.Encode(uri, qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR code: %w", err)
	}

	secretB32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)

	return &TOTPSetupResult{
		Secret:    secretB32,
		QRCodeURI: uri,
		QRCodePNG: png,
	}, nil
}

// VerifySetup はセットアップ時の初回 TOTP コード検証を行い、MFA を有効化する。
// session が非 nil の場合、セットアップ完了後にセッションの MFA 状態も更新する（ログインフロー内セットアップ）。
func (s *MFATOTPService) VerifySetup(ctx context.Context, userID uuid.UUID, code string, session *model.Session) error {
	mfaConfig, err := s.mfaConfigFinder.FindByUserIDAndType(ctx, userID, "totp")
	if err != nil {
		return fmt.Errorf("failed to find MFA config: %w", err)
	}
	if mfaConfig == nil {
		return ErrMFANotConfigured
	}
	if mfaConfig.Enabled {
		return ErrMFAAlreadyConfigured
	}
	if mfaConfig.TotpConfig == nil {
		return ErrMFANotConfigured
	}

	// シークレット復号
	secret, err := s.decrypt(mfaConfig.TotpConfig.SecretKeyEncrypted, s.encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt TOTP secret: %w", err)
	}

	// TOTP 検証
	step, valid := totp.Validate(
		secret, code, time.Now(),
		int(mfaConfig.TotpConfig.Period),
		int(mfaConfig.TotpConfig.Digits),
		mfaConfig.TotpConfig.LastUsedStep,
	)
	if !valid {
		return ErrInvalidTOTPCode
	}

	// last_used_step 更新
	if err := s.totpConfigStore.UpdateLastUsedStep(ctx, mfaConfig.TotpConfig.ID, step); err != nil {
		return fmt.Errorf("failed to update last used step: %w", err)
	}

	// MFA 有効化
	now := time.Now()
	mfaConfig.Enabled = true
	mfaConfig.VerifiedAt = &now
	if err := s.mfaConfigStore.Update(ctx, mfaConfig); err != nil {
		return fmt.Errorf("failed to enable MFA config: %w", err)
	}

	// ログインフロー内セットアップの場合、セッションの MFA 状態も更新
	if session != nil && session.PendingMFA {
		amr := model.StringSlice{"pwd", "otp"}
		acr := "urn:mace:incommon:iap:silver"
		if err := s.sessionUpdater.UpdateMFACompleted(ctx, session.ID, amr, acr); err != nil {
			return fmt.Errorf("failed to update session MFA status: %w", err)
		}
	}

	return nil
}

// VerifyLogin はログインフローでの TOTP 検証を行い、セッションの MFA 状態を更新する。
func (s *MFATOTPService) VerifyLogin(ctx context.Context, session *model.Session, code string) error {
	if !session.PendingMFA {
		return ErrMFANotPending
	}

	mfaConfig, err := s.mfaConfigFinder.FindEnabledByUserID(ctx, session.UserID)
	if err != nil {
		return fmt.Errorf("failed to find MFA config: %w", err)
	}
	if mfaConfig == nil || mfaConfig.TotpConfig == nil {
		return ErrMFANotConfigured
	}

	// シークレット復号
	secret, err := s.decrypt(mfaConfig.TotpConfig.SecretKeyEncrypted, s.encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt TOTP secret: %w", err)
	}

	// TOTP 検証
	step, valid := totp.Validate(
		secret, code, time.Now(),
		int(mfaConfig.TotpConfig.Period),
		int(mfaConfig.TotpConfig.Digits),
		mfaConfig.TotpConfig.LastUsedStep,
	)
	if !valid {
		return ErrInvalidTOTPCode
	}

	// last_used_step 更新
	if err := s.totpConfigStore.UpdateLastUsedStep(ctx, mfaConfig.TotpConfig.ID, step); err != nil {
		return fmt.Errorf("failed to update last used step: %w", err)
	}

	// セッション更新: pending_mfa=false, AMR=["pwd","otp"], ACR=silver
	amr := model.StringSlice{"pwd", "otp"}
	acr := "urn:mace:incommon:iap:silver"
	if err := s.sessionUpdater.UpdateMFACompleted(ctx, session.ID, amr, acr); err != nil {
		return fmt.Errorf("failed to update session MFA status: %w", err)
	}

	return nil
}

// IsEnabled はユーザーが有効な MFA 設定を持っているかを返す。
func (s *MFATOTPService) IsEnabled(ctx context.Context, userID uuid.UUID) (bool, error) {
	config, err := s.mfaConfigFinder.FindEnabledByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	return config != nil, nil
}

// Disable はユーザーの TOTP MFA 設定を無効化（削除）する。
func (s *MFATOTPService) Disable(ctx context.Context, userID uuid.UUID) error {
	mfaConfig, err := s.mfaConfigFinder.FindByUserIDAndType(ctx, userID, "totp")
	if err != nil {
		return fmt.Errorf("failed to find MFA config: %w", err)
	}
	if mfaConfig == nil {
		return ErrMFANotConfigured
	}

	// CASCADE で TotpConfig も削除される
	if err := s.mfaConfigStore.Delete(ctx, mfaConfig.ID); err != nil {
		return fmt.Errorf("failed to delete MFA config: %w", err)
	}

	return nil
}
