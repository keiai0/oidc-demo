SET search_path TO op;

CREATE TABLE initial_access_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash VARCHAR(512) NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    max_registrations INT NOT NULL DEFAULT 0,
    used_count INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_initial_access_tokens_tenant ON initial_access_tokens(tenant_id);
CREATE INDEX idx_initial_access_tokens_hash ON initial_access_tokens(token_hash);

COMMENT ON TABLE initial_access_tokens IS 'Dynamic Client Registration (RFC 7591) で使用する Initial Access Token';
COMMENT ON COLUMN initial_access_tokens.token_hash IS 'トークンの SHA256 ハッシュ（平文は発行時のみ返却）';
COMMENT ON COLUMN initial_access_tokens.tenant_id IS '発行先テナント';
COMMENT ON COLUMN initial_access_tokens.max_registrations IS '最大登録回数（0 = 無制限）';
COMMENT ON COLUMN initial_access_tokens.used_count IS '使用済み登録回数';
COMMENT ON COLUMN initial_access_tokens.expires_at IS 'トークン有効期限';
COMMENT ON COLUMN initial_access_tokens.revoked_at IS '無効化日時（NULL = 有効）';
