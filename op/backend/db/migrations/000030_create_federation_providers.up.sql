CREATE TABLE IF NOT EXISTS federation_providers (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id),
    name             VARCHAR(63) NOT NULL,
    issuer           VARCHAR(2048) NOT NULL,
    client_id        VARCHAR(255) NOT NULL,
    client_secret_enc TEXT NOT NULL,
    scopes           VARCHAR(1024) NOT NULL DEFAULT 'openid profile email',
    auto_provision   BOOLEAN NOT NULL DEFAULT true,
    status           VARCHAR(31) NOT NULL DEFAULT 'active',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_federation_providers_tenant_id ON federation_providers(tenant_id);

COMMENT ON TABLE federation_providers IS '外部 IdP 連携プロバイダ設定。テナントごとに複数の外部 OIDC Provider を登録可能';
COMMENT ON COLUMN federation_providers.name IS 'プロバイダ識別名（URL パスに使用）: google, azure_ad 等';
COMMENT ON COLUMN federation_providers.issuer IS '外部 IdP の issuer URL（Discovery に使用）';
COMMENT ON COLUMN federation_providers.client_id IS '外部 IdP に登録した OAuth2 client_id';
COMMENT ON COLUMN federation_providers.client_secret_enc IS '外部 IdP の client_secret（AES-256-GCM 暗号化）';
COMMENT ON COLUMN federation_providers.scopes IS '外部 IdP に要求するスコープ（スペース区切り）';
COMMENT ON COLUMN federation_providers.auto_provision IS 'JIT プロビジョニング: 初回ログイン時にユーザーを自動作成するか';
