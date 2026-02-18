-- =============================================================================
-- 開発用シードデータ
-- 本番環境では実行しないこと
-- =============================================================================

-- 開発用テナント
INSERT INTO tenants (id, code, name) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'demo', 'Demo Tenant')
ON CONFLICT (id) DO NOTHING;

-- 開発用ユーザー (testuser)
INSERT INTO users (id, tenant_id, login_id, email, email_verified, name, status) VALUES
    ('b0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'testuser', 'testuser@example.com', true, 'Test User', 'active')
ON CONFLICT (id) DO NOTHING;

-- 開発用クライアント (demo-rp)
INSERT INTO clients (id, tenant_id, client_id, client_secret_hash, name, grant_types, response_types, token_endpoint_auth_method, require_pkce, status) VALUES
    ('c0000000-0000-0000-0000-000000000001',
     'a0000000-0000-0000-0000-000000000001',
     'demo-rp',
     '$argon2id$v=19$m=65536,t=3,p=4$XLnZ4+fz/MCzO+Ax4vynLg$wb2a0Uwr1mgjZnTMCFylw7XCCgBR81ueDM+OmWcGQGM',
     'Demo RP',
     '["authorization_code", "refresh_token"]',
     '["code"]',
     'client_secret_post',
     true,
     'active')
ON CONFLICT (id) DO NOTHING;

INSERT INTO redirect_uris (client_id, uri) VALUES
    ('c0000000-0000-0000-0000-000000000001', 'http://localhost:3001/api/auth/callback')
ON CONFLICT DO NOTHING;

INSERT INTO post_logout_redirect_uris (client_id, uri) VALUES
    ('c0000000-0000-0000-0000-000000000001', 'http://localhost:3001')
ON CONFLICT DO NOTHING;

-- testuser のパスワードクレデンシャル (password: "password")
INSERT INTO credentials (id, user_id, type, created_at, updated_at) VALUES
    ('d0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001', 'password', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO password_credentials (id, credential_id, password_hash, algorithm, updated_at) VALUES
    ('e0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000001', '$argon2id$v=19$m=65536,t=3,p=4$p/1zKj9TNYyP56xhmjyAtQ$JXv9c0nSybiSzZ3goGEpvciL2MHEAimZaBcuzYXxQdc', 'argon2id', NOW())
ON CONFLICT (id) DO NOTHING;

-- 開発用管理ユーザー (admin / admin)
INSERT INTO admin_users (id, login_id, password_hash, name, status) VALUES
    ('f0000000-0000-0000-0000-000000000001', 'admin', '$argon2id$v=19$m=65536,t=3,p=4$Uo9ePSD5eq6LtwxkBckU7Q$IfMdE7Ae3M+KxlgYyAFouY5jVeoZ7q4XOM7ZkYQoSdg', 'Administrator', 'active')
ON CONFLICT (id) DO NOTHING;
