CREATE TABLE IF NOT EXISTS op.email_change_tokens (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID         NOT NULL REFERENCES op.users(id) ON DELETE CASCADE,
    new_email  VARCHAR(255) NOT NULL,
    token_hash VARCHAR(512) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ  NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_change_tokens_user_id ON op.email_change_tokens(user_id);

COMMENT ON TABLE op.email_change_tokens IS 'メールアドレス変更の確認トークン（24時間有効、1回使い切り）';
COMMENT ON COLUMN op.email_change_tokens.user_id IS '変更リクエストを行ったユーザーID';
COMMENT ON COLUMN op.email_change_tokens.new_email IS '変更先の新しいメールアドレス';
COMMENT ON COLUMN op.email_change_tokens.token_hash IS 'トークンの SHA-256 ハッシュ（平文はメールで送信のみ）';
COMMENT ON COLUMN op.email_change_tokens.expires_at IS 'トークン有効期限（発行から24時間）';
COMMENT ON COLUMN op.email_change_tokens.used_at IS '使用済み時刻。非NULLは使用済み';
