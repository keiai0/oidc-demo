CREATE TABLE IF NOT EXISTS credentials (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        VARCHAR(63) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_credentials_user_id ON credentials(user_id);

COMMENT ON TABLE credentials IS 'ユーザーの認証手段（ポリモーフィック設計）。複数の認証方式に対応するための親テーブル';
COMMENT ON COLUMN credentials.type IS '認証方式の種別: password / oidc_provider';
