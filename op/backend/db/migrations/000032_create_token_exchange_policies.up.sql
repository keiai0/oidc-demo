CREATE TABLE IF NOT EXISTS op.token_exchange_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES op.clients(id) ON DELETE CASCADE,
    allowed_subject_token_types JSONB NOT NULL DEFAULT '["urn:ietf:params:oauth:token-type:access_token"]',
    allowed_requested_token_types JSONB NOT NULL DEFAULT '["urn:ietf:params:oauth:token-type:access_token"]',
    allowed_audiences JSONB NOT NULL DEFAULT '[]',
    allow_impersonation BOOLEAN NOT NULL DEFAULT false,
    allow_delegation BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_token_exchange_policies_client UNIQUE (client_id)
);

COMMENT ON TABLE op.token_exchange_policies IS 'Token Exchange (RFC 8693) ポリシー。クライアントごとに1レコード';
COMMENT ON COLUMN op.token_exchange_policies.client_id IS 'ポリシー対象のクライアント (1:1)';
COMMENT ON COLUMN op.token_exchange_policies.allowed_subject_token_types IS '受け入れる subject_token_type の URI リスト';
COMMENT ON COLUMN op.token_exchange_policies.allowed_requested_token_types IS '発行可能な requested_token_type の URI リスト';
COMMENT ON COLUMN op.token_exchange_policies.allowed_audiences IS '許可される audience 値のリスト（空配列=制限なし）';
COMMENT ON COLUMN op.token_exchange_policies.allow_impersonation IS 'Impersonation（なりすまし: actor_token なし）を許可するか';
COMMENT ON COLUMN op.token_exchange_policies.allow_delegation IS 'Delegation（委任: act クレーム付与）を許可するか';
