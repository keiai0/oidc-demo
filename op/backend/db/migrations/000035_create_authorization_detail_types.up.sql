CREATE TABLE IF NOT EXISTS op.authorization_detail_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES op.tenants(id) ON DELETE CASCADE,
    type_name VARCHAR(255) NOT NULL,
    description TEXT,
    json_schema JSONB,
    allowed_actions JSONB NOT NULL DEFAULT '[]',
    allowed_locations JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_authorization_detail_types_tenant_type UNIQUE (tenant_id, type_name)
);

COMMENT ON TABLE op.authorization_detail_types IS 'Rich Authorization Requests (RFC 9396) の認可詳細タイプ定義。テナントごとにサポートする type を登録';
COMMENT ON COLUMN op.authorization_detail_types.tenant_id IS '所属テナント';
COMMENT ON COLUMN op.authorization_detail_types.type_name IS 'authorization_details の type フィールド値 (RFC 9396 Section 2)';
COMMENT ON COLUMN op.authorization_detail_types.description IS 'タイプの説明（管理画面・同意画面表示用）';
COMMENT ON COLUMN op.authorization_detail_types.json_schema IS 'タイプ固有フィールドのバリデーション用 JSON Schema';
COMMENT ON COLUMN op.authorization_detail_types.allowed_actions IS '許可される actions の値リスト（空配列=制限なし）';
COMMENT ON COLUMN op.authorization_detail_types.allowed_locations IS '許可される locations の値リスト（空配列=制限なし）';
