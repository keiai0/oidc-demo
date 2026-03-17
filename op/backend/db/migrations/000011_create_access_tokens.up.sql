CREATE TABLE IF NOT EXISTS access_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    jti         VARCHAR(255) NOT NULL UNIQUE,
    session_id  UUID REFERENCES sessions(id),
    client_id   UUID NOT NULL REFERENCES clients(id),
    scope       VARCHAR(1024) NOT NULL,
    dpop_jkt    VARCHAR(255),
    claims_param TEXT,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ
);

CREATE INDEX idx_access_tokens_jti ON access_tokens(jti);

COMMENT ON TABLE access_tokens IS 'アクセストークンの発行記録。JWT形式で発行するが、失効管理のためにDBにも記録する';
COMMENT ON COLUMN access_tokens.jti IS 'JWT ID クレーム。トークンの一意識別子';
COMMENT ON COLUMN access_tokens.session_id IS 'セッションID。client_credentials grant の場合は NULL';
COMMENT ON COLUMN access_tokens.scope IS '付与されたスコープ';
COMMENT ON COLUMN access_tokens.dpop_jkt IS 'DPoP JWK Thumbprint (cnf.jkt)。NULLの場合はBearerトークン (RFC 9449)';
COMMENT ON COLUMN access_tokens.claims_param IS 'OIDC claims リクエストパラメータ (Section 5.5)。userinfo での個別クレーム要求に使用';
COMMENT ON COLUMN access_tokens.revoked_at IS '失効日時。/revoke で設定';
