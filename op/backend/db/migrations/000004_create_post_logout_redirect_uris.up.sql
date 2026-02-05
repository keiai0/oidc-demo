CREATE TABLE IF NOT EXISTS post_logout_redirect_uris (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id   UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    uri         VARCHAR(2048) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_post_logout_redirect_uris_client_id ON post_logout_redirect_uris(client_id);

COMMENT ON TABLE post_logout_redirect_uris IS 'ログアウト後のリダイレクト先URI。RP-Initiated Logout 1.0 で使用';
COMMENT ON COLUMN post_logout_redirect_uris.uri IS 'ログアウト後リダイレクトURI';
