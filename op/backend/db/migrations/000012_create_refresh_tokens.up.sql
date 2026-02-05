CREATE TABLE IF NOT EXISTS refresh_tokens (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash          VARCHAR(255) NOT NULL UNIQUE,
    parent_id           UUID REFERENCES refresh_tokens(id),
    session_id          UUID NOT NULL REFERENCES sessions(id),
    access_token_id     UUID NOT NULL REFERENCES access_tokens(id),
    expires_at          TIMESTAMPTZ NOT NULL,
    revoked_at          TIMESTAMPTZ,
    reuse_detected_at   TIMESTAMPTZ
);

CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_session_id ON refresh_tokens(session_id);

COMMENT ON TABLE refresh_tokens IS 'リフレッシュトークン。Rotation + Reuse Detection (RFC 9700) を実装。トークン本体はハッシュ化して保存（DB流出対策）';
COMMENT ON COLUMN refresh_tokens.token_hash IS 'トークンのハッシュ値';
COMMENT ON COLUMN refresh_tokens.parent_id IS 'ローテーション元トークン。Reuse Detection 用';
COMMENT ON COLUMN refresh_tokens.session_id IS '再利用検知時にセッション全体を失効させるために必要';
COMMENT ON COLUMN refresh_tokens.access_token_id IS '対応するアクセストークン';
COMMENT ON COLUMN refresh_tokens.reuse_detected_at IS '再利用検知日時（監査用）';
