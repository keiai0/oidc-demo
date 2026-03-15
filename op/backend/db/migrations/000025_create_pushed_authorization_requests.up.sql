CREATE TABLE IF NOT EXISTS pushed_authorization_requests (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_uri VARCHAR(255) NOT NULL UNIQUE,
    client_id   UUID NOT NULL REFERENCES clients(id),
    parameters  JSONB NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ
);

CREATE INDEX idx_par_request_uri ON pushed_authorization_requests(request_uri);

COMMENT ON TABLE pushed_authorization_requests IS 'Pushed Authorization Request (RFC 9126)。認可リクエストパラメータをバックチャネルで事前送信する';
COMMENT ON COLUMN pushed_authorization_requests.request_uri IS 'PAR のリクエスト URI (urn:ietf:params:oauth:request_uri:...)';
COMMENT ON COLUMN pushed_authorization_requests.parameters IS '認可リクエストパラメータ (JSON)';
COMMENT ON COLUMN pushed_authorization_requests.expires_at IS '有効期限 (デフォルト60秒)';
COMMENT ON COLUMN pushed_authorization_requests.used_at IS '使用済み日時。二重使用防止';
