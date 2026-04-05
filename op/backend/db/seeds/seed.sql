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
INSERT INTO clients (id, client_id, client_secret_hash, name, grant_types, response_types, token_endpoint_auth_method, require_pkce, status) VALUES
    ('c0000000-0000-0000-0000-000000000001',
     'demo-rp',
     '$argon2id$v=19$m=65536,t=3,p=4$XLnZ4+fz/MCzO+Ax4vynLg$wb2a0Uwr1mgjZnTMCFylw7XCCgBR81ueDM+OmWcGQGM',
     'Demo RP',
     '["authorization_code", "refresh_token"]',
     '["code"]',
     'client_secret_post',
     true,
     'active')
ON CONFLICT (id) DO NOTHING;

-- テナント-クライアント紐づけ
INSERT INTO tenant_clients (tenant_id, client_id) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001')
ON CONFLICT (tenant_id, client_id) DO NOTHING;

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

-- CSRF デモ用攻撃者アカウント (attacker / password)
INSERT INTO users (id, tenant_id, login_id, email, email_verified, name, status) VALUES
    ('b0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'attacker', 'attacker@example.com', true, 'Attacker User', 'active')
ON CONFLICT (id) DO NOTHING;

INSERT INTO credentials (id, user_id, type, created_at, updated_at) VALUES
    ('d0000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000002', 'password', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- パスワード: "password" (testuser と同じハッシュ)
INSERT INTO password_credentials (id, credential_id, password_hash, algorithm, updated_at) VALUES
    ('e0000000-0000-0000-0000-000000000002', 'd0000000-0000-0000-0000-000000000002', '$argon2id$v=19$m=65536,t=3,p=4$p/1zKj9TNYyP56xhmjyAtQ$JXv9c0nSybiSzZ3goGEpvciL2MHEAimZaBcuzYXxQdc', 'argon2id', NOW())
ON CONFLICT (id) DO NOTHING;

-- Token Exchange デモ用 M2M クライアント (demo-service / demo-service-secret)
INSERT INTO clients (id, client_id, client_secret_hash, name, grant_types, response_types, token_endpoint_auth_method, require_pkce, status) VALUES
    ('c0000000-0000-0000-0000-000000000002',
     'demo-service',
     '$argon2id$v=19$m=65536,t=3,p=4$XLnZ4+fz/MCzO+Ax4vynLg$wb2a0Uwr1mgjZnTMCFylw7XCCgBR81ueDM+OmWcGQGM',
     'Demo Service (Token Exchange)',
     '["client_credentials", "urn:ietf:params:oauth:grant-type:token-exchange"]',
     '["code"]',
     'client_secret_basic',
     false,
     'active')
ON CONFLICT (id) DO NOTHING;

-- demo-service のテナント紐づけ
INSERT INTO tenant_clients (tenant_id, client_id) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000002')
ON CONFLICT (tenant_id, client_id) DO NOTHING;

-- Token Exchange ポリシー (demo-service: Impersonation + Delegation 両方許可)
INSERT INTO token_exchange_policies (id, client_id, allowed_subject_token_types, allowed_requested_token_types, allowed_audiences, allow_impersonation, allow_delegation) VALUES
    ('f1000000-0000-0000-0000-000000000001',
     'c0000000-0000-0000-0000-000000000002',
     '["urn:ietf:params:oauth:token-type:access_token"]',
     '["urn:ietf:params:oauth:token-type:access_token"]',
     '[]',
     true,
     true)
ON CONFLICT (client_id) DO NOTHING;

-- Rich Authorization Requests (RFC 9396) デモ用認可詳細タイプ
INSERT INTO authorization_detail_types (id, tenant_id, type_name, description, allowed_actions, allowed_locations) VALUES
    ('f2000000-0000-0000-0000-000000000001',
     'a0000000-0000-0000-0000-000000000001',
     'payment_initiation',
     '送金指示: 指定口座から送金を開始する権限',
     '["initiate", "status"]',
     '[]')
ON CONFLICT (tenant_id, type_name) DO NOTHING;

INSERT INTO authorization_detail_types (id, tenant_id, type_name, description, allowed_actions, allowed_locations) VALUES
    ('f2000000-0000-0000-0000-000000000002',
     'a0000000-0000-0000-0000-000000000001',
     'account_information',
     '口座情報照会: 口座の残高・取引履歴を閲覧する権限',
     '["read", "list"]',
     '["https://example.com/accounts"]')
ON CONFLICT (tenant_id, type_name) DO NOTHING;

-- 開発用管理ユーザー (admin / admin)
INSERT INTO admin_users (id, login_id, password_hash, name, status) VALUES
    ('f0000000-0000-0000-0000-000000000001', 'admin', '$argon2id$v=19$m=65536,t=3,p=4$Uo9ePSD5eq6LtwxkBckU7Q$IfMdE7Ae3M+KxlgYyAFouY5jVeoZ7q4XOM7ZkYQoSdg', 'Administrator', 'active')
ON CONFLICT (id) DO NOTHING;
