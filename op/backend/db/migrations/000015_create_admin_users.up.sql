SET search_path TO op;

CREATE TABLE IF NOT EXISTS admin_users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login_id      VARCHAR(255) NOT NULL,
    password_hash TEXT         NOT NULL,
    name          VARCHAR(255) NOT NULL,
    status        VARCHAR(31)  NOT NULL DEFAULT 'active',
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_admin_users_login_id UNIQUE (login_id)
);

COMMENT ON TABLE admin_users IS '管理画面のログインユーザー。OIDC ユーザー (users) とは完全に分離。テナントに依存しない';
COMMENT ON COLUMN admin_users.login_id IS 'ログインID（グローバルで一意）';
COMMENT ON COLUMN admin_users.password_hash IS 'argon2id ハッシュ済みパスワード';
COMMENT ON COLUMN admin_users.name IS '表示名';
COMMENT ON COLUMN admin_users.status IS 'active / disabled';
COMMENT ON COLUMN admin_users.last_login_at IS '最終ログイン日時';
