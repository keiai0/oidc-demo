SET search_path TO op;

CREATE TABLE client_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id) UNIQUE,
    registration_access_token_hash VARCHAR(512) NOT NULL,
    registration_client_uri VARCHAR(2048) NOT NULL,
    software_id VARCHAR(255),
    software_version VARCHAR(255),
    initial_access_token_id UUID REFERENCES initial_access_tokens(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_client_registrations_client ON client_registrations(client_id);

COMMENT ON TABLE client_registrations IS 'Dynamic Client Registration (RFC 7591/7592) で登録されたクライアントのメタデータ';
COMMENT ON COLUMN client_registrations.client_id IS '登録されたクライアント（clients テーブル FK）';
COMMENT ON COLUMN client_registrations.registration_access_token_hash IS 'Registration Access Token の SHA256 ハッシュ（RFC 7592 認証用）';
COMMENT ON COLUMN client_registrations.registration_client_uri IS 'クライアント設定エンドポイント URI（RFC 7592）';
COMMENT ON COLUMN client_registrations.software_id IS 'ソフトウェア識別子（RFC 7591 Section 2）';
COMMENT ON COLUMN client_registrations.software_version IS 'ソフトウェアバージョン（RFC 7591 Section 2）';
COMMENT ON COLUMN client_registrations.initial_access_token_id IS '登録に使用された IAT（追跡用）';
