CREATE TABLE IF NOT EXISTS clients (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                   UUID NOT NULL REFERENCES tenants(id),
    client_id                   VARCHAR(255) NOT NULL UNIQUE,
    client_secret_hash          VARCHAR(512) NOT NULL,
    name                        VARCHAR(255) NOT NULL,
    grant_types                 JSONB NOT NULL DEFAULT '["authorization_code"]',
    response_types              JSONB NOT NULL DEFAULT '["code"]',
    token_endpoint_auth_method  VARCHAR(63) NOT NULL DEFAULT 'client_secret_basic',
    require_pkce                BOOLEAN NOT NULL DEFAULT true,
    frontchannel_logout_uri     VARCHAR(2048),
    backchannel_logout_uri      VARCHAR(2048),
    status                      VARCHAR(31) NOT NULL DEFAULT 'active',
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clients_tenant_id ON clients(tenant_id);

COMMENT ON TABLE clients IS 'OPに登録されたOIDCクライアント（RP）。OIDC仕様の Relying Party に対応';
COMMENT ON COLUMN clients.tenant_id IS '所属テナント';
COMMENT ON COLUMN clients.client_id IS 'OIDC仕様の client_id';
COMMENT ON COLUMN clients.client_secret_hash IS 'client_secret のハッシュ値';
COMMENT ON COLUMN clients.name IS 'クライアント表示名';
COMMENT ON COLUMN clients.grant_types IS '許可する認可フロー';
COMMENT ON COLUMN clients.response_types IS '許可するレスポンスタイプ';
COMMENT ON COLUMN clients.token_endpoint_auth_method IS 'トークンエンドポイントの認証方式';
COMMENT ON COLUMN clients.require_pkce IS 'PKCE必須フラグ（S256のみサポート）';
COMMENT ON COLUMN clients.frontchannel_logout_uri IS 'Front-Channel Logout 通知先';
COMMENT ON COLUMN clients.backchannel_logout_uri IS 'Back-Channel Logout 通知先';
COMMENT ON COLUMN clients.status IS 'active / disabled';
