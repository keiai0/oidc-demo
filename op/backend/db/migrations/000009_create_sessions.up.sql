CREATE TABLE IF NOT EXISTS sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    ip_address  VARCHAR(45) NOT NULL,
    user_agent  TEXT NOT NULL DEFAULT '',
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user_id_revoked_at ON sessions(user_id, revoked_at);

COMMENT ON TABLE sessions IS 'ユーザーのログインセッション。OIDC "sid" クレームに対応。SLO・セッション管理に使用。Redis ではなく DB 管理（expires_at で TTL 制御）';
COMMENT ON COLUMN sessions.id IS 'OIDC "sid" クレームとして使用';
COMMENT ON COLUMN sessions.ip_address IS '接続元IP（IPv6対応）';
COMMENT ON COLUMN sessions.user_agent IS 'ブラウザ情報';
COMMENT ON COLUMN sessions.expires_at IS 'セッション有効期限';
COMMENT ON COLUMN sessions.revoked_at IS '失効日時。SLO・強制ログアウト時に設定';
