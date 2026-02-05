CREATE TABLE IF NOT EXISTS authorization_codes (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id              UUID NOT NULL REFERENCES sessions(id),
    client_id               UUID NOT NULL REFERENCES clients(id),
    code                    VARCHAR(255) NOT NULL UNIQUE,
    redirect_uri            VARCHAR(2048) NOT NULL,
    scope                   VARCHAR(1024) NOT NULL,
    nonce                   VARCHAR(255),
    code_challenge          VARCHAR(255),
    code_challenge_method   VARCHAR(31),
    expires_at              TIMESTAMPTZ NOT NULL,
    used_at                 TIMESTAMPTZ
);

CREATE INDEX idx_authorization_codes_code ON authorization_codes(code);

COMMENT ON TABLE authorization_codes IS '認可コード（使い捨て）。RFC 6749 Section 4.1.2: 認可コードフローで発行される一時コード';
COMMENT ON COLUMN authorization_codes.code IS '認可コード本体';
COMMENT ON COLUMN authorization_codes.redirect_uri IS '発行時の redirect_uri。トークン発行時に一致検証する';
COMMENT ON COLUMN authorization_codes.scope IS '要求されたスコープ';
COMMENT ON COLUMN authorization_codes.nonce IS 'OIDC nonce。IDトークンのリプレイ攻撃防止';
COMMENT ON COLUMN authorization_codes.code_challenge IS 'PKCE code_challenge';
COMMENT ON COLUMN authorization_codes.code_challenge_method IS 'PKCE メソッド（S256のみ）';
COMMENT ON COLUMN authorization_codes.used_at IS '使用済み日時。null=未使用。再利用は MUST 拒否';
