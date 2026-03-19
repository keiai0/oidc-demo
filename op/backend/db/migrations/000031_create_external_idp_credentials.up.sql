CREATE TABLE IF NOT EXISTS external_idp_credentials (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    credential_id    UUID NOT NULL REFERENCES credentials(id) ON DELETE CASCADE,
    provider_id      UUID NOT NULL REFERENCES federation_providers(id),
    provider_subject VARCHAR(255) NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(provider_id, provider_subject)
);

CREATE INDEX idx_external_idp_credentials_credential_id ON external_idp_credentials(credential_id);
CREATE INDEX idx_external_idp_credentials_provider_subject ON external_idp_credentials(provider_id, provider_subject);

COMMENT ON TABLE external_idp_credentials IS '外部 IdP 経由の認証クレデンシャル。credentials テーブルの子テーブル（type=oidc_provider）';
COMMENT ON COLUMN external_idp_credentials.provider_id IS '連携先の federation_provider';
COMMENT ON COLUMN external_idp_credentials.provider_subject IS '外部 IdP が発行した sub クレーム値';
