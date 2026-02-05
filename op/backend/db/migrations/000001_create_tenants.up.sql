CREATE TABLE IF NOT EXISTS tenants (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                    VARCHAR(63) NOT NULL UNIQUE,
    name                    VARCHAR(255) NOT NULL,
    session_lifetime        INT NOT NULL DEFAULT 3600,
    auth_code_lifetime      INT NOT NULL DEFAULT 60,
    access_token_lifetime   INT NOT NULL DEFAULT 3600,
    refresh_token_lifetime  INT NOT NULL DEFAULT 2592000,
    id_token_lifetime       INT NOT NULL DEFAULT 3600,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE tenants IS 'マルチテナントの単位（組織・企業）。テナントごとにトークン有効期限等のセキュリティポリシーを設定できる';
COMMENT ON COLUMN tenants.code IS 'URLに含めるテナント識別子 (例: /{tenant_code}/authorize)';
COMMENT ON COLUMN tenants.name IS 'テナント表示名';
COMMENT ON COLUMN tenants.session_lifetime IS '認証セッション有効期限（秒）';
COMMENT ON COLUMN tenants.auth_code_lifetime IS '認可コード有効期限（秒）。RFC 6749: 最大10分推奨';
COMMENT ON COLUMN tenants.access_token_lifetime IS 'アクセストークン有効期限（秒）';
COMMENT ON COLUMN tenants.refresh_token_lifetime IS 'リフレッシュトークン有効期限（秒）。デフォルト30日';
COMMENT ON COLUMN tenants.id_token_lifetime IS 'IDトークン有効期限（秒）';
