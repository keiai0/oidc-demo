package management

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// AdminUserFinder は認証用の管理者ユーザー検索を定義する。
type AdminUserFinder interface {
	// FindByLoginID はログイン ID で管理者ユーザーを検索する。見つからない場合は (nil, nil) を返す。
	FindByLoginID(ctx context.Context, loginID string) (*model.AdminUser, error)
	// FindByID は UUID で管理者ユーザーを検索する。見つからない場合は (nil, nil) を返す。
	FindByID(ctx context.Context, id uuid.UUID) (*model.AdminUser, error)
	// UpdateLastLoginAt は最終ログイン日時を更新する。
	UpdateLastLoginAt(ctx context.Context, id uuid.UUID, t time.Time) error
}

// AdminSessionStore は管理者セッションの永続化を管理する。
type AdminSessionStore interface {
	// Create は新しい管理者セッションを永続化する。
	Create(ctx context.Context, session *model.AdminSession) error
	// FindByID は UUID で管理者セッションを検索する。見つからない場合は (nil, nil) を返す。
	FindByID(ctx context.Context, id uuid.UUID) (*model.AdminSession, error)
	// Revoke は管理者セッションを失効済みにする。
	Revoke(ctx context.Context, id uuid.UUID) error
}

// PasswordVerifyFunc は平文パスワードをハッシュと照合する。
type PasswordVerifyFunc func(password, hash string) (bool, error)
