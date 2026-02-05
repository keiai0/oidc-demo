CREATE TABLE IF NOT EXISTS id_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    jti         VARCHAR(255) NOT NULL UNIQUE,
    session_id  UUID NOT NULL REFERENCES sessions(id),
    client_id   UUID NOT NULL REFERENCES clients(id),
    nonce       VARCHAR(255),
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE id_tokens IS 'IDトークンの発行履歴。検証・監査用。JWT形式で発行';
COMMENT ON COLUMN id_tokens.jti IS 'JWT ID クレーム';
COMMENT ON COLUMN id_tokens.nonce IS '認可リクエスト時の nonce';
