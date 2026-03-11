CREATE TABLE IF NOT EXISTS op.mfa_configs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES op.users(id),
    type        VARCHAR(31) NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, type)
);

COMMENT ON TABLE op.mfa_configs IS 'ユーザーのMFA設定（認証方式ごとに1レコード）';
COMMENT ON COLUMN op.mfa_configs.user_id IS 'ユーザーID';
COMMENT ON COLUMN op.mfa_configs.type IS 'MFA種別（totp / webauthn）';
COMMENT ON COLUMN op.mfa_configs.enabled IS 'MFAが有効化されているか';
COMMENT ON COLUMN op.mfa_configs.verified_at IS '初回検証（セットアップ完了）時刻';
