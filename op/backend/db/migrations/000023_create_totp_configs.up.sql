CREATE TABLE IF NOT EXISTS op.totp_configs (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mfa_config_id        UUID NOT NULL REFERENCES op.mfa_configs(id) ON DELETE CASCADE,
    secret_key_encrypted TEXT NOT NULL,
    algorithm            VARCHAR(15) NOT NULL DEFAULT 'SHA1',
    digits               SMALLINT NOT NULL DEFAULT 6,
    period               SMALLINT NOT NULL DEFAULT 30,
    last_used_step       BIGINT,
    UNIQUE(mfa_config_id)
);

COMMENT ON TABLE op.totp_configs IS 'TOTP設定（mfa_configsの子テーブル）';
COMMENT ON COLUMN op.totp_configs.mfa_config_id IS 'MFA設定ID';
COMMENT ON COLUMN op.totp_configs.secret_key_encrypted IS 'AES-256-GCMで暗号化されたTOTPシークレット';
COMMENT ON COLUMN op.totp_configs.algorithm IS 'HMACアルゴリズム（SHA1）';
COMMENT ON COLUMN op.totp_configs.digits IS 'TOTPコード桁数';
COMMENT ON COLUMN op.totp_configs.period IS 'TOTPステップ間隔（秒）';
COMMENT ON COLUMN op.totp_configs.last_used_step IS 'リプレイ攻撃防止用の最終使用ステップ番号';
