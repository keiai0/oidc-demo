CREATE TABLE IF NOT EXISTS sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id),
    tenant_id           UUID NOT NULL REFERENCES tenants(id),
    ip_address          VARCHAR(45) NOT NULL,
    user_agent          TEXT NOT NULL DEFAULT '',
    auth_time           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    amr                 JSONB NOT NULL DEFAULT '["pwd"]',
    acr                 VARCHAR(255) NOT NULL DEFAULT 'urn:mace:incommon:iap:bronze',
    pending_mfa         BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_setup_required  BOOLEAN NOT NULL DEFAULT FALSE,
    webauthn_challenge   TEXT,
    expires_at          TIMESTAMPTZ NOT NULL,
    revoked_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user_id_revoked_at ON sessions(user_id, revoked_at);

COMMENT ON TABLE sessions IS 'ユーザーのログインセッション。OIDC "sid" クレームに対応。SLO・セッション管理に使用。Redis ではなく DB 管理（expires_at で TTL 制御）';
COMMENT ON COLUMN sessions.id IS 'OIDC "sid" クレームとして使用';
COMMENT ON COLUMN sessions.ip_address IS '接続元IP（IPv6対応）';
COMMENT ON COLUMN sessions.user_agent IS 'ブラウザ情報';
COMMENT ON COLUMN sessions.auth_time IS 'ユーザー認証が行われた時刻（OIDC auth_time）';
COMMENT ON COLUMN sessions.amr IS '認証方法の参照（JSON配列: ["pwd"], ["pwd","otp"]）';
COMMENT ON COLUMN sessions.acr IS '認証コンテキストクラス参照';
COMMENT ON COLUMN sessions.pending_mfa IS 'MFA検証待ちフラグ（trueの場合、パスワード認証は完了しているがMFA未完了）';
COMMENT ON COLUMN sessions.mfa_setup_required IS 'MFA セットアップが必要か（テナント強制 + MFA 未設定時に true）';
COMMENT ON COLUMN sessions.webauthn_challenge IS 'WebAuthn チャレンジデータ（JSON）。登録・認証フローの begin〜complete 間で一時保存';
COMMENT ON COLUMN sessions.expires_at IS 'セッション有効期限';
COMMENT ON COLUMN sessions.revoked_at IS '失効日時。SLO・強制ログアウト時に設定';
