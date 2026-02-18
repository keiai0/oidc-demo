SET search_path TO op;

CREATE TABLE IF NOT EXISTS admin_sessions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_user_id UUID        NOT NULL REFERENCES admin_users(id),
    ip_address    VARCHAR(45) NOT NULL,
    user_agent    TEXT        NOT NULL DEFAULT '',
    expires_at    TIMESTAMPTZ NOT NULL,
    revoked_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_sessions_admin_user_id ON admin_sessions(admin_user_id);

COMMENT ON TABLE admin_sessions IS '管理画面のログインセッション。OIDC セッション (sessions) とは分離';
COMMENT ON COLUMN admin_sessions.admin_user_id IS '対応する管理ユーザー';
COMMENT ON COLUMN admin_sessions.ip_address IS '接続元IP（IPv6対応、最大45文字）';
COMMENT ON COLUMN admin_sessions.user_agent IS 'ブラウザ情報';
COMMENT ON COLUMN admin_sessions.expires_at IS 'セッション有効期限';
COMMENT ON COLUMN admin_sessions.revoked_at IS '明示的な失効日時（ログアウト等）';
