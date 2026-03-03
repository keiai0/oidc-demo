CREATE TABLE IF NOT EXISTS tenant_clients (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    client_id   UUID NOT NULL REFERENCES clients(id),
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, client_id)
);

CREATE INDEX idx_tenant_clients_tenant_id ON tenant_clients(tenant_id);
CREATE INDEX idx_tenant_clients_client_id ON tenant_clients(client_id);

COMMENT ON TABLE tenant_clients IS 'テナントとクライアントの多対多関連';
COMMENT ON COLUMN tenant_clients.tenant_id IS '関連テナント';
COMMENT ON COLUMN tenant_clients.client_id IS '関連クライアント';
COMMENT ON COLUMN tenant_clients.enabled IS 'このテナントでのクライアント利用可否';
