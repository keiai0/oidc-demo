CREATE TABLE IF NOT EXISTS dpop_jti_cache (
    jti        VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dpop_jti_cache_created_at ON dpop_jti_cache(created_at);

COMMENT ON TABLE dpop_jti_cache IS 'DPoP proof JWT の JTI リプレイ防止キャッシュ (RFC 9449)';
COMMENT ON COLUMN dpop_jti_cache.jti IS 'DPoP proof の JWT ID。一意性を保証しリプレイ攻撃を防ぐ';
