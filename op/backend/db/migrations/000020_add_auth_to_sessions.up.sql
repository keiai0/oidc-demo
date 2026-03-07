ALTER TABLE op.sessions
  ADD COLUMN auth_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  ADD COLUMN amr JSONB NOT NULL DEFAULT '["pwd"]',
  ADD COLUMN acr VARCHAR(255) NOT NULL DEFAULT 'urn:mace:incommon:iap:bronze';

COMMENT ON COLUMN op.sessions.auth_time IS 'ユーザー認証が行われた時刻（OIDC auth_time）';
COMMENT ON COLUMN op.sessions.amr IS '認証方法の参照（JSON配列: ["pwd"], ["pwd","otp"]）';
COMMENT ON COLUMN op.sessions.acr IS '認証コンテキストクラス参照';
