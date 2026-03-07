# テナントモデル移行計画: モデル1 → モデル2

## 概要

現在のテナントモデル（Client が Tenant に 1:n で所属）を、Client と Tenant（Organization）が n:n で関連するモデルに移行する。

## 前提条件

- OIDC コアフロー（Phase 1）が一通り動作していること
- Phase 2（管理機能）が完成していること
- Phase 3（RP動作検証）が完成していること

これらが完了した後に本計画を実施する。

---

## 現状のモデル（モデル1）

```
Tenant 1:n Client（clients.tenant_id で直接参照）
Tenant 1:n User（users.tenant_id で直接参照）
```

- Client は 1 つの Tenant にのみ所属する
- テナント間のデータは完全に分離されている

## 移行後のモデル（モデル2）

```
Client（独立エンティティ）
Tenant（Organization、独立エンティティ）
Client n:n Tenant（中間テーブル tenant_clients で関連）
Tenant 1:n User（変更なし）
```

- Client はテナントから独立して存在する
- 1 つの Client を複数の Tenant が利用できる
- Tenant ごとにどの Client を利用可能かを管理する

---

## DB スキーマ変更

### 1. 中間テーブルの追加

```sql
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
COMMENT ON COLUMN tenant_clients.enabled IS 'このテナントでのクライアント利用可否';
```

### 2. clients テーブルから tenant_id を削除

```sql
-- 既存データの移行
INSERT INTO tenant_clients (tenant_id, client_id)
SELECT tenant_id, id FROM clients;

-- tenant_id カラムの削除
ALTER TABLE clients DROP COLUMN tenant_id;
```

### 3. 認可コード・トークンへの影響

Client がテナントに直接紐づかなくなるが、認可フローのテナント情報は **`sessions.tenant_id`** 経由で取得可能。
既存の `sessions.tenant_id` で十分であり、追加のスキーマ変更は不要。

---

## バックエンド変更

### Model 層

| ファイル | 変更内容 |
|---------|---------|
| `model/client.go` | `TenantID` フィールドを削除、`Tenant` リレーション定義を削除 |
| `model/tenant_client.go` | 新規: `TenantClient` エンティティ |

### Store 層

| ファイル | 変更内容 |
|---------|---------|
| `store/client_repository.go` | `ListByTenantID` → 中間テーブル JOIN に変更、`Create` から `tenant_id` を除去 |
| `store/tenant_client_repository.go` | 新規: 中間テーブル CRUD（追加・削除・存在確認） |

### OIDC フロー

| 処理 | 変更内容 |
|-----|---------|
| 認可エンドポイント (`authorize.go`) | `client.TenantID != tenant.ID` チェックを `tenant_clients` テーブルの存在確認に変更 |

トークンエンドポイント (`token_authcode.go`, `token_refresh.go`) は変更不要。
テナント情報は `authCode.Session.TenantID` 経由で取得しており、`client.TenantID` を参照していない。

### Management API

| エンドポイント | 変更内容 |
|--------------|---------|
| `POST /management/v1/tenants/:id/clients` | クライアント作成 + 中間テーブルへの関連追加 |
| `GET /management/v1/tenants/:id/clients` | 中間テーブル JOIN で取得 |
| `GET /management/v1/clients` | 新規: 全クライアント一覧（テナント横断） |
| `POST /management/v1/clients/:id/tenants` | 新規: クライアントにテナントを紐づけ |
| `DELETE /management/v1/clients/:id/tenants/:tenant_id` | 新規: クライアントからテナント紐づけを解除 |

`management/client.go` の `clientResponse` から `tenant_id` フィールドを削除し、代わりに `tenants` フィールド（関連テナント一覧）を追加する。

### deps.go インターフェース

| パッケージ | 変更内容 |
|-----------|---------|
| `oidc/deps.go` | `ClientFinder` は変更なし（`FindByClientID` はテナント非依存で問題ない） |
| `oidc/deps.go` | `TenantClientChecker` インターフェース追加: テナントでクライアントが利用可能か確認 |
| `management/deps.go` | `ClientStore.ListByTenantID` は維持（内部実装が JOIN に変更） |
| `management/deps.go` | `TenantClientStore` インターフェース追加 |

---

## フロントエンド変更

### 型定義

| ファイル | 変更内容 |
|---------|---------|
| `types/client.ts` | `Client` 型から `tenant_id: string` を削除、`tenants?: TenantSummary[]` を追加 |

### ページ変更

| ページ | 変更内容 |
|-------|---------|
| クライアント一覧 (`/management/clients`) | N+1 クエリ（全テナント→各テナントのクライアント取得）を `GET /management/v1/clients` 単一 API に置き換え |
| クライアント作成 (`tenants/detail/clients/new`) | テナント経由の作成は維持（作成と同時にテナント紐づけ） |
| クライアント詳細 (`/management/clients/detail`) | 「関連テナント」セクション追加（テナントの追加・解除 UI）、削除後のリダイレクト先をグローバルクライアント一覧に変更 |

### API 層

| ファイル | 変更内容 |
|---------|---------|
| `lib/api/clients.ts` | `listAll()` 追加、`addTenant()` / `removeTenant()` 追加 |

---

## Seed データ更新

```sql
-- 移行後の seed: clients から tenant_id を除去
INSERT INTO clients (id, client_id, client_secret_hash, name, grant_types, response_types,
    token_endpoint_auth_method, require_pkce, status) VALUES
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

-- テナント-クライアント関連
INSERT INTO tenant_clients (tenant_id, client_id) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001')
ON CONFLICT DO NOTHING;
```

---

## マイグレーション手順

1. `tenant_clients` テーブルを作成（up マイグレーション）
2. 既存の `clients.tenant_id` データを `tenant_clients` に移行（同一マイグレーション内）
3. バックエンド（Model / Store / OIDC / Management API）を中間テーブル対応に更新
4. **テスト・ビルド確認**（`go test ./...` / `go build ./...`）
5. `clients.tenant_id` カラムを削除（別のマイグレーション）
6. seed データを更新
7. フロントエンドを更新
8. **E2E 確認**（RP からのログインフロー全体が動作すること）

---

## 影響範囲

### 影響が大きい箇所
- 認可エンドポイント（テナント×クライアントの検証ロジック）
- クライアント管理 API（CRUD 全般）
- フロントエンドのクライアント関連ページ

### 影響がない箇所
- ユーザー管理（`users.tenant_id` は変更なし）
- セッション管理（`sessions.tenant_id` は変更なし）
- トークンエンドポイント（テナント情報は `Session.TenantID` 経由で取得）
- 署名鍵管理
- JWT 生成・検証ロジック
- Discovery / JWKS エンドポイント
- `auth/service.go`（ログイン処理。テナントは URL の `tenant_code` から取得）
- `auth/me.go`（セッション情報返却。`Session.TenantID` を使用）
- `RevokeByTenantID`（session / access_token / refresh_token。`Session.TenantID` を使用）

---

## リスクと対策

| リスク | 対策 |
|-------|-----|
| 既存の OIDC フローが壊れる | 移行前に E2E テスト（Phase 3 の RP）で動作を確認し、移行後に再確認 |
| データ移行漏れ | up/down マイグレーションをペアで作成し、ロールバック可能にする |
| 認可時のテナント特定が曖昧になる | 認可エンドポイントの URL に `tenant_code` が含まれるため、テナント特定は既存の仕組みを維持 |
| カラム削除前にバックエンドが壊れる | マイグレーションを2段階に分割: (1) 中間テーブル追加+データ移行 → (2) カラム削除。間にテスト挟む |
