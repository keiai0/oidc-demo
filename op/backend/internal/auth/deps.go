package auth

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

type TenantFinder interface {
	FindByCode(ctx context.Context, code string) (*model.Tenant, error)
}

type UserFinder interface {
	FindByTenantAndLoginID(ctx context.Context, tenantID uuid.UUID, loginID string) (*model.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	UpdateLastLoginAt(ctx context.Context, id uuid.UUID, t time.Time) error
}

type SessionStore interface {
	Create(ctx context.Context, session *model.Session) error
	FindByID(ctx context.Context, id uuid.UUID) (*model.Session, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

// LoginAttemptTracker はログイン試行の追跡操作を定義する。
type LoginAttemptTracker interface {
	// IncrementFailedLogin は連続失敗回数をインクリメントし、閾値でロックする。
	IncrementFailedLogin(ctx context.Context, userID uuid.UUID) error
	// ResetFailedLogin は連続失敗回数とロック状態をリセットする。
	ResetFailedLogin(ctx context.Context, userID uuid.UUID) error
}

// PasswordHistoryStore はパスワード履歴の永続化操作を定義する。
type PasswordHistoryStore interface {
	// Create はパスワード履歴レコードを作成する。
	Create(ctx context.Context, history *model.PasswordHistory) error
	// FindRecentByUserID はユーザーの直近 limit 件のパスワード履歴を返す。
	FindRecentByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]model.PasswordHistory, error)
}

// PasswordUpdater はパスワードクレデンシャルの更新操作を定義する。
type PasswordUpdater interface {
	// UpdateHash はパスワードクレデンシャルのハッシュを更新する。
	UpdateHash(ctx context.Context, credentialID uuid.UUID, newHash string) error
}

// UserFinderWithCredentials はクレデンシャル付きのユーザー検索を定義する。
type UserFinderWithCredentials interface {
	FindByIDWithCredentials(ctx context.Context, id uuid.UUID) (*model.User, error)
}

// PasswordResetTokenStore はパスワードリセットトークンの永続化操作を定義する。
type PasswordResetTokenStore interface {
	// Create はパスワードリセットトークンを作成する。
	Create(ctx context.Context, token *model.PasswordResetToken) error
	// FindByTokenHash はトークンハッシュでリセットトークンを検索する。
	FindByTokenHash(ctx context.Context, hash string) (*model.PasswordResetToken, error)
	// MarkAsUsed はトークンを使用済みにする。
	MarkAsUsed(ctx context.Context, id uuid.UUID) error
	// InvalidateByUserID はユーザーの未使用リセットトークンを全て無効化する。
	InvalidateByUserID(ctx context.Context, userID uuid.UUID) error
}

// UserFinderByEmail はメールアドレスとテナントでユーザーを検索する。
type UserFinderByEmail interface {
	FindByTenantAndEmail(ctx context.Context, tenantID uuid.UUID, email string) (*model.User, error)
}

// EmailSender はメール送信操作を定義する。
type EmailSender interface {
	SendPasswordResetEmail(ctx context.Context, email, token string) error
	// SendEmailChangeEmail は変更先メールアドレス宛に確認メールを送信する。
	SendEmailChangeEmail(ctx context.Context, newEmail, token string) error
}

// EmailChangeTokenStore はメールアドレス変更トークンの永続化操作を定義する。
type EmailChangeTokenStore interface {
	// Create はメールアドレス変更トークンを作成する。
	Create(ctx context.Context, token *model.EmailChangeToken) error
	// FindByTokenHash はトークンハッシュで変更トークンを検索する。
	FindByTokenHash(ctx context.Context, hash string) (*model.EmailChangeToken, error)
	// MarkAsUsed はトークンを使用済みにする。
	MarkAsUsed(ctx context.Context, id uuid.UUID) error
	// InvalidateByUserID はユーザーの未使用変更トークンを全て無効化する。
	InvalidateByUserID(ctx context.Context, userID uuid.UUID) error
}

// UserEmailUpdater はユーザーのメールアドレス更新を定義する。
type UserEmailUpdater interface {
	// UpdateEmail はメールアドレスを更新し email_verified を true にする。
	UpdateEmail(ctx context.Context, userID uuid.UUID, newEmail string) error
}

// BackupCodeStore は MFA バックアップコードの永続化操作を定義する。
type BackupCodeStore interface {
	// Create はバックアップコードを作成する。
	Create(ctx context.Context, code *model.BackupCode) error
	// FindUnusedByUserID はユーザーの未使用バックアップコード一覧を返す。
	FindUnusedByUserID(ctx context.Context, userID uuid.UUID) ([]model.BackupCode, error)
	// MarkAsUsed はバックアップコードを使用済みにする。
	MarkAsUsed(ctx context.Context, id uuid.UUID) error
	// DeleteByUserID はユーザーのバックアップコードを全て削除する。
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

// SessionLister はユーザーのアクティブセッション一覧取得を定義する。
type SessionLister interface {
	// FindActiveByUserID はユーザーのアクティブセッション一覧を返す。
	FindActiveByUserID(ctx context.Context, userID uuid.UUID) ([]model.Session, error)
}

type PasswordVerifyFunc func(password, hash string) (bool, error)
type HashPasswordFunc func(password string) (string, error)
