CREATE TABLE IF NOT EXISTS op.user_consents (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES op.users(id),
    client_id  UUID NOT NULL REFERENCES op.clients(id),
    scopes     VARCHAR(1024) NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, client_id)
);

COMMENT ON TABLE op.user_consents IS 'ユーザーのスコープ同意記録';
COMMENT ON COLUMN op.user_consents.scopes IS 'スペース区切りの許可済みスコープ';
