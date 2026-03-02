package auth

import (
	"context"

	"github.com/google/uuid"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// MfaConfigFinder は MFA 設定の検索操作を定義する。
type MfaConfigFinder interface {
	// FindEnabledByUserID はユーザーの有効な MFA 設定を TotpConfig 付きで返す。
	FindEnabledByUserID(ctx context.Context, userID uuid.UUID) (*model.MfaConfig, error)
	// FindByUserIDAndType はユーザーの指定タイプの MFA 設定を TotpConfig 付きで返す。
	FindByUserIDAndType(ctx context.Context, userID uuid.UUID, mfaType string) (*model.MfaConfig, error)
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

// EncryptFunc はデータを暗号化する関数型。
type EncryptFunc func(plaintext []byte, key []byte) (string, error)

// DecryptFunc はデータを復号する関数型。
type DecryptFunc func(encrypted string, key []byte) ([]byte, error)
