package management

import (
	"context"

	"github.com/google/uuid"
	"github.com/isurugi-k/oidc-demo/op/backend/internal/model"
)

// TenantStore は管理機能向けのテナント永続化操作を定義する。
type TenantStore interface {
	// List はテナントをページネーション付きで返す。(tenants, totalCount, error) を返す。
	List(ctx context.Context, limit, offset int) ([]model.Tenant, int64, error)
	// Create は新しいテナントを永続化する。
	Create(ctx context.Context, tenant *model.Tenant) error
	// FindByID は UUID でテナントを検索する。見つからない場合は (nil, nil) を返す。
	FindByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error)
	// FindByCode はコードでテナントを検索する。見つからない場合は (nil, nil) を返す。
	FindByCode(ctx context.Context, code string) (*model.Tenant, error)
	// Update はテナントの変更を保存する。
	Update(ctx context.Context, tenant *model.Tenant) error
}

// ClientStore は管理機能向けのクライアント永続化操作を定義する。
type ClientStore interface {
	// ListByTenantID はテナントに属するクライアントをページネーション付きで返す。
	ListByTenantID(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]model.Client, int64, error)
	// Create は新しいクライアントを永続化する。
	Create(ctx context.Context, client *model.Client) error
	// FindByID は UUID でクライアントを検索する。
	FindByID(ctx context.Context, id uuid.UUID) (*model.Client, error)
	// FindByIDWithRelations はリダイレクト URI をプリロードしてクライアントを検索する。
	FindByIDWithRelations(ctx context.Context, id uuid.UUID) (*model.Client, error)
	// Update はクライアントの変更を保存する。
	Update(ctx context.Context, client *model.Client) error
	// UpdateSecretHash は client_secret_hash フィールドのみを更新する。
	UpdateSecretHash(ctx context.Context, id uuid.UUID, hash string) error
	// SoftDelete はクライアントの status を "disabled" に設定して論理削除する。
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

// RedirectURIStore はリダイレクト URI の永続化操作を定義する。
type RedirectURIStore interface {
	// ListByClientID はクライアントに属する全てのリダイレクト URI を返す。
	ListByClientID(ctx context.Context, clientDBID uuid.UUID) ([]model.RedirectURI, error)
	// Create は新しいリダイレクト URI を永続化する。
	Create(ctx context.Context, uri *model.RedirectURI) error
	// FindByID は UUID でリダイレクト URI を検索する。
	FindByID(ctx context.Context, id uuid.UUID) (*model.RedirectURI, error)
	// Delete はリダイレクト URI を削除する。
	Delete(ctx context.Context, id uuid.UUID) error
}

// PostLogoutRedirectURIStore はポストログアウトリダイレクト URI の永続化操作を定義する。
type PostLogoutRedirectURIStore interface {
	// ListByClientID はクライアントに属する全てのポストログアウトリダイレクト URI を返す。
	ListByClientID(ctx context.Context, clientDBID uuid.UUID) ([]model.PostLogoutRedirectURI, error)
	// Create は新しいポストログアウトリダイレクト URI を永続化する。
	Create(ctx context.Context, uri *model.PostLogoutRedirectURI) error
	// Delete はポストログアウトリダイレクト URI を削除する。
	Delete(ctx context.Context, id uuid.UUID) error
}

// SignKeyStore は管理機能向けの署名鍵永続化操作を定義する。
type SignKeyStore interface {
	// FindAll は有効・無効を問わず全ての署名鍵を返す。
	FindAll(ctx context.Context) ([]model.SignKey, error)
	// FindByKID は鍵 ID で署名鍵を検索する。
	FindByKID(ctx context.Context, kid string) (*model.SignKey, error)
	// Deactivate は鍵を無効化する。
	Deactivate(ctx context.Context, kid string) error
	// FindAllActive は全ての有効な署名鍵を返す。
	FindAllActive(ctx context.Context) ([]model.SignKey, error)
}

// SessionRevoker はセッションの一括失効操作を定義する。
type SessionRevoker interface {
	// RevokeAll は全ての有効なセッションを失効させる。影響行数を返す。
	RevokeAll(ctx context.Context) (int64, error)
	// RevokeByTenantID はテナントに属する全ての有効なセッションを失効させる。
	RevokeByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error)
	// RevokeByUserID はユーザーに属する全ての有効なセッションを失効させる。
	RevokeByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}

// AccessTokenRevoker はアクセストークンの一括失効操作を定義する。
type AccessTokenRevoker interface {
	// RevokeAll は全ての有効なアクセストークンを失効させる。
	RevokeAll(ctx context.Context) (int64, error)
	// RevokeByTenantID はテナントに属する全ての有効なアクセストークンを失効させる。
	RevokeByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error)
	// RevokeByUserID はユーザーに属する全ての有効なアクセストークンを失効させる。
	RevokeByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}

// RefreshTokenRevoker はリフレッシュトークンの一括失効操作を定義する。
type RefreshTokenRevoker interface {
	// RevokeAll は全ての有効なリフレッシュトークンを失効させる。
	RevokeAll(ctx context.Context) (int64, error)
	// RevokeByTenantID はテナントに属する全ての有効なリフレッシュトークンを失効させる。
	RevokeByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, error)
	// RevokeByUserID はユーザーに属する全ての有効なリフレッシュトークンを失効させる。
	RevokeByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}

// HashPasswordFunc は平文パスワードを argon2id でハッシュ化する。
type HashPasswordFunc func(password string) (string, error)

// KeyRotator は新しい署名鍵を生成し、既存の鍵を無効化する。
type KeyRotator interface {
	// RotateKey は新しい有効な署名鍵を作成し、既存の有効な鍵を全て無効化する。
	RotateKey(ctx context.Context) (*model.SignKey, error)
}
