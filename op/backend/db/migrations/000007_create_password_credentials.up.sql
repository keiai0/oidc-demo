CREATE TABLE IF NOT EXISTS password_credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    credential_id   UUID NOT NULL UNIQUE REFERENCES credentials(id) ON DELETE CASCADE,
    password_hash   VARCHAR(512) NOT NULL,
    algorithm       VARCHAR(63) NOT NULL DEFAULT 'argon2id',
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE password_credentials IS 'パスワード認証情報。credentials の子テーブル（type = password）';
COMMENT ON COLUMN password_credentials.password_hash IS 'argon2id ハッシュ値';
COMMENT ON COLUMN password_credentials.algorithm IS 'ハッシュアルゴリズム';
