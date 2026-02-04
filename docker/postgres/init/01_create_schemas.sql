-- =============================================================================
-- OP / RP スキーマ・ロール作成
-- テーブル作成はマイグレーション（golang-migrate / Drizzle）が担う
-- =============================================================================

-- OP用ロール
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'op_user') THEN
        CREATE ROLE op_user WITH LOGIN PASSWORD 'op_password';
    END IF;
END
$$;

-- RP用ロール
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'rp_user') THEN
        CREATE ROLE rp_user WITH LOGIN PASSWORD 'rp_password';
    END IF;
END
$$;

-- OPスキーマ
CREATE SCHEMA IF NOT EXISTS op AUTHORIZATION op_user;
GRANT ALL PRIVILEGES ON SCHEMA op TO op_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA op GRANT ALL ON TABLES TO op_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA op GRANT ALL ON SEQUENCES TO op_user;

-- RPスキーマ
CREATE SCHEMA IF NOT EXISTS rp AUTHORIZATION rp_user;
GRANT ALL PRIVILEGES ON SCHEMA rp TO rp_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA rp GRANT ALL ON TABLES TO rp_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA rp GRANT ALL ON SEQUENCES TO rp_user;
