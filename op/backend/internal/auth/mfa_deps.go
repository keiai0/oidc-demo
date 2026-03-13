package auth

import (
	"context"

	"github.com/google/uuid"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// MfaConfigFinder は MFA 設定の検索操作を定義する。
type MfaConfigFinder interface {
	// FindEnabledByUserID はユーザーの有効な MFA 設定を返す（最初に見つかった1件）。
	FindEnabledByUserID(ctx context.Context, userID uuid.UUID) (*model.MfaConfig, error)
	// FindByUserIDAndType はユーザーの指定タイプの MFA 設定を返す。
	FindByUserIDAndType(ctx context.Context, userID uuid.UUID, mfaType string) (*model.MfaConfig, error)
	// FindAllEnabledByUserID はユーザーの全ての有効な MFA 設定を返す（複数方式対応）。
	FindAllEnabledByUserID(ctx context.Context, userID uuid.UUID) ([]model.MfaConfig, error)
}

// MfaConfigStore は MFA 設定の永続化操作を定義する。
type MfaConfigStore interface {
	Create(ctx context.Context, config *model.MfaConfig) error
	Update(ctx context.Context, config *model.MfaConfig) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// TotpConfigStore は TOTP 設定の永続化操作を定義する。
type TotpConfigStore interface {
	Create(ctx context.Context, config *model.TotpConfig) error
	FindByMfaConfigID(ctx context.Context, mfaConfigID uuid.UUID) (*model.TotpConfig, error)
	UpdateLastUsedStep(ctx context.Context, id uuid.UUID, step int64) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SessionUpdater は MFA 完了後のセッション更新操作を定義する。
type SessionUpdater interface {
	UpdateMFACompleted(ctx context.Context, sessionID uuid.UUID, amr model.StringSlice, acr string) error
}

// WebAuthnCredentialStore は WebAuthn クレデンシャルの永続化操作を定義する。
type WebAuthnCredentialStore interface {
	// Create は WebAuthn クレデンシャルを作成する。
	Create(ctx context.Context, cred *model.WebAuthnCredential) error
	// FindByMfaConfigID は指定 MFA 設定の全 WebAuthn クレデンシャルを返す。
	FindByMfaConfigID(ctx context.Context, mfaConfigID uuid.UUID) ([]model.WebAuthnCredential, error)
	// FindByCredentialID は credential_id で WebAuthn クレデンシャルを検索する。
	FindByCredentialID(ctx context.Context, credentialID string) (*model.WebAuthnCredential, error)
	// UpdateSignCount は sign_count を更新する。
	UpdateSignCount(ctx context.Context, id uuid.UUID, signCount uint32) error
	// Delete は WebAuthn クレデンシャルを削除する。
	Delete(ctx context.Context, id uuid.UUID) error
}

// WebAuthnChallengeUpdater は WebAuthn チャレンジのセッション保存操作を定義する。
type WebAuthnChallengeUpdater interface {
	// UpdateWebAuthnChallenge は WebAuthn チャレンジデータをセッションに保存する。nil で消去。
	UpdateWebAuthnChallenge(ctx context.Context, sessionID uuid.UUID, challenge *string) error
}

// EncryptFunc はデータを暗号化する関数型。
type EncryptFunc func(plaintext []byte, key []byte) (string, error)

// DecryptFunc はデータを復号する関数型。
type DecryptFunc func(encrypted string, key []byte) ([]byte, error)
