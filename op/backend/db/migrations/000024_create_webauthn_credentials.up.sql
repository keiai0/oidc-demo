CREATE TABLE IF NOT EXISTS webauthn_credentials (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mfa_config_id    UUID NOT NULL REFERENCES mfa_configs(id) ON DELETE CASCADE,
    credential_id    TEXT NOT NULL UNIQUE,
    public_key       TEXT NOT NULL,
    attestation_type TEXT NOT NULL DEFAULT '',
    aaguid           TEXT NOT NULL DEFAULT '',
    sign_count       INT NOT NULL DEFAULT 0,
    backup_eligible  BOOLEAN NOT NULL DEFAULT FALSE,
    backup_state     BOOLEAN NOT NULL DEFAULT FALSE,
    name             VARCHAR(255) NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webauthn_credentials_mfa_config_id ON webauthn_credentials(mfa_config_id);

COMMENT ON TABLE webauthn_credentials IS 'WebAuthn（パスキー）のクレデンシャル。1ユーザーが複数デバイスを登録可能';
COMMENT ON COLUMN webauthn_credentials.credential_id IS 'WebAuthn credential ID（base64url エンコード済み）';
COMMENT ON COLUMN webauthn_credentials.public_key IS 'CBOR エンコードされた公開鍵（base64）';
COMMENT ON COLUMN webauthn_credentials.attestation_type IS 'Attestation の種類（none, packed 等）';
COMMENT ON COLUMN webauthn_credentials.aaguid IS 'Authenticator の AAGUID';
COMMENT ON COLUMN webauthn_credentials.sign_count IS '署名カウンター（クローン検知用）';
COMMENT ON COLUMN webauthn_credentials.name IS 'ユーザーが付けたデバイス名';
