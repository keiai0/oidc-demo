CREATE TABLE IF NOT EXISTS redirect_uris (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id   UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    uri         VARCHAR(2048) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_redirect_uris_client_id ON redirect_uris(client_id);

COMMENT ON TABLE redirect_uris IS 'クライアントの認可コールバックURI。RFC 6749 Section 3.1.2.3: 完全一致で検証する';
COMMENT ON COLUMN redirect_uris.uri IS 'コールバックURI';
