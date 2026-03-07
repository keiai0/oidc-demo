ALTER TABLE op.users
  ADD COLUMN failed_login_count SMALLINT NOT NULL DEFAULT 0,
  ADD COLUMN locked_until TIMESTAMPTZ;

COMMENT ON COLUMN op.users.failed_login_count IS '連続ログイン失敗回数';
COMMENT ON COLUMN op.users.locked_until IS 'ロック解除時刻。NULLはロックなし';
