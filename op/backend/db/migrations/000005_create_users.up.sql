CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    login_id        VARCHAR(255) NOT NULL,
    email           VARCHAR(255) NOT NULL,
    email_verified  BOOLEAN NOT NULL DEFAULT false,
    name            VARCHAR(255),
    status          VARCHAR(31) NOT NULL DEFAULT 'active',
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, login_id)
);

CREATE INDEX idx_users_email ON users(email);

COMMENT ON TABLE users IS 'OPが管理するユーザー。OIDC標準クレームと認証に必要な属性のみ持つ。RP固有の業務属性はRPが sub をキーに自前管理する';
COMMENT ON COLUMN users.id IS 'OIDC "sub" クレームとして使用';
COMMENT ON COLUMN users.tenant_id IS '所属テナント';
COMMENT ON COLUMN users.login_id IS 'テナント内で一意のログインID';
COMMENT ON COLUMN users.email IS 'メールアドレス（OIDC email クレーム）';
COMMENT ON COLUMN users.email_verified IS 'メール確認済みフラグ（OIDC email_verified クレーム）';
COMMENT ON COLUMN users.name IS '表示名（OIDC name クレーム, profile スコープ）';
COMMENT ON COLUMN users.status IS 'active / locked / disabled';
COMMENT ON COLUMN users.last_login_at IS '最終ログイン日時';
