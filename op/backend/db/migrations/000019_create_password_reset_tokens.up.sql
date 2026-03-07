CREATE TABLE IF NOT EXISTS op.password_reset_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES op.users(id),
    token_hash VARCHAR(512) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_reset_tokens_user_id ON op.password_reset_tokens(user_id);

COMMENT ON TABLE op.password_reset_tokens IS 'パスワードリセットトークン';
COMMENT ON COLUMN op.password_reset_tokens.token_hash IS 'トークンの SHA-256 ハッシュ（平文はメールで送信のみ）';
COMMENT ON COLUMN op.password_reset_tokens.used_at IS '使用済み時刻。非NULLは使用済み';
