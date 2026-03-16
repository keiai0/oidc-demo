CREATE TABLE IF NOT EXISTS sign_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kid             VARCHAR(255) NOT NULL UNIQUE,
    algorithm       VARCHAR(31) NOT NULL DEFAULT 'RS256',
    public_key      TEXT NOT NULL,
    private_key_ref TEXT NOT NULL,
    status          VARCHAR(31) NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rotated_at      TIMESTAMPTZ
);

CREATE INDEX idx_sign_keys_kid ON sign_keys(kid);
CREATE INDEX idx_sign_keys_status ON sign_keys(status);

COMMENT ON TABLE sign_keys IS 'JWT署名用の鍵ペア。鍵ローテーション対応。秘密鍵は AES-256-GCM で暗号化して保存';
COMMENT ON COLUMN sign_keys.kid IS 'JWT header の "kid" クレーム';
COMMENT ON COLUMN sign_keys.algorithm IS '署名アルゴリズム';
COMMENT ON COLUMN sign_keys.public_key IS 'PEM形式の公開鍵。JWKSエンドポイントで公開';
COMMENT ON COLUMN sign_keys.private_key_ref IS 'AES-256-GCM で暗号化された秘密鍵';
COMMENT ON COLUMN sign_keys.status IS '鍵のライフサイクル状態: active（署名+検証）/ passive（検証のみ, JWKS に含む）/ expired（削除対象）';
COMMENT ON COLUMN sign_keys.rotated_at IS 'active→passive への移行日時。null=現役';
