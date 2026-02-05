CREATE TABLE IF NOT EXISTS password_histories (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    password_hash   VARCHAR(512) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_histories_user_id ON password_histories(user_id);

COMMENT ON TABLE password_histories IS 'パスワード履歴。再利用防止用';
COMMENT ON COLUMN password_histories.password_hash IS '過去のパスワードハッシュ';
