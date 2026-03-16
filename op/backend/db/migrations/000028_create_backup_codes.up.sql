CREATE TABLE IF NOT EXISTS op.backup_codes (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID         NOT NULL REFERENCES op.users(id) ON DELETE CASCADE,
    code_hash  VARCHAR(512) NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_backup_codes_user_id ON op.backup_codes(user_id);

COMMENT ON TABLE op.backup_codes IS 'MFAバックアップコード（10個、argon2idハッシュ保存、1回使い切り）';
COMMENT ON COLUMN op.backup_codes.user_id IS '所有ユーザーID';
COMMENT ON COLUMN op.backup_codes.code_hash IS 'バックアップコードの argon2id ハッシュ（平文は生成時1回限り返却）。argon2idは同一平文でも毎回異なるハッシュを生成するため UNIQUE 制約なし';
COMMENT ON COLUMN op.backup_codes.used_at IS '使用済み時刻。非NULLは使用済み';
