CREATE TABLE IF NOT EXISTS device_authorization_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    client_id       UUID NOT NULL REFERENCES clients(id),
    device_code     VARCHAR(255) UNIQUE NOT NULL,
    user_code       VARCHAR(15) UNIQUE NOT NULL,
    scope           VARCHAR(1024) NOT NULL DEFAULT '',
    status          VARCHAR(31) NOT NULL DEFAULT 'pending',
    session_id      UUID REFERENCES sessions(id),
    poll_interval   INT NOT NULL DEFAULT 5,
    expires_at      TIMESTAMPTZ NOT NULL,
    last_polled_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_device_auth_requests_device_code ON device_authorization_requests(device_code);
CREATE INDEX idx_device_auth_requests_user_code ON device_authorization_requests(user_code);
CREATE INDEX idx_device_auth_requests_expires_at ON device_authorization_requests(expires_at);

COMMENT ON TABLE device_authorization_requests IS 'Device Authorization Grant (RFC 8628) のデバイス認可リクエスト';
COMMENT ON COLUMN device_authorization_requests.device_code IS 'デバイスが token endpoint へのポーリングに使用する高エントロピーコード';
COMMENT ON COLUMN device_authorization_requests.user_code IS 'ユーザーが認証画面で入力する短いコード（BCDF-GHJK 形式）';
COMMENT ON COLUMN device_authorization_requests.status IS 'リクエスト状態: pending / approved / denied / expired';
COMMENT ON COLUMN device_authorization_requests.session_id IS '承認時にリンクされるユーザーセッション';
COMMENT ON COLUMN device_authorization_requests.poll_interval IS 'デバイスのポーリング間隔（秒）。slow_down 時に +5s される';
