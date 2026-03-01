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

### 3. 認可コード・トークンにテナント情報を追加

Client がテナントに紐づかなくなるため、認可フローのどのテナント経由でアクセスされたかをトークンに記録する必要がある。

```sql
-- authorization_codes にテナント情報を追加（session 経由で取得可能だが明示化）
-- sessions.tenant_id は既に存在するため追加不要
```

---

## バックエンド変更

### Store 層

| ファイル | 変更内容 |
|---------|---------|
| `store/client.go` | `ListByTenantID` → 中間テーブル JOIN に変更 |
| `store/client.go` | `Create` から `tenant_id` を除去 |
| `store/tenant_client.go` | 新規: 中間テーブル CRUD |

### Model 層

| ファイル | 変更内容 |
|---------|---------|
| `model/client.go` | `TenantID` フィールドを削除 |
| `model/tenant_client.go` | 新規: `TenantClient` エンティティ |

### Management API

| エンドポイント | 変更内容 |
|--------------|---------|
| `POST /management/v1/tenants/:id/clients` | クライアント作成 + 中間テーブルへの関連追加 |
| `GET /management/v1/tenants/:id/clients` | 中間テーブル JOIN で取得 |
| `POST /management/v1/clients` | 新規: テナント非依存のクライアント作成 |
| `GET /management/v1/clients` | 新規: 全クライアント一覧 |
| `POST /management/v1/clients/:id/tenants` | 新規: クライアントにテナントを紐づけ |
| `DELETE /management/v1/clients/:id/tenants/:tenant_id` | 新規: クライアントからテナント紐づけを解除 |

### OIDC フロー

| 処理 | 変更内容 |
|-----|---------|
| 認可エンドポイント | `client_id` の検索時に `tenant_clients` を確認し、該当テナントで利用可能か検証 |
| トークンエンドポイント | クライアント認証時のテナント検証を中間テーブル経由に変更 |

---

## フロントエンド変更

### ルーティング

| 現在 | 移行後 |
|-----|-------|
| テナント詳細 → クライアント一覧 | そのまま維持（テナントに紐づくクライアントを表示） |
| — | クライアント一覧（全件）ページは既に存在 |
| — | クライアント詳細にテナント紐づけ管理 UI を追加 |

### ページ変更

| ページ | 変更内容 |
|-------|---------|
| クライアント一覧 (`/management/clients`) | API がテナント横断で返すようになるため、フロントエンドでの集約処理を API 呼び出しに置き換え |
| クライアント作成 | テナント選択を必須ではなくオプションに変更（後からテナントを紐づけ可能） |
| クライアント詳細 | 「関連テナント」セクション追加（テナントの追加・解除 UI） |

---

## マイグレーション手順

1. `tenant_clients` テーブルを作成
2. 既存の `clients.tenant_id` データを `tenant_clients` に移行
3. バックエンド API を中間テーブル対応に更新
4. フロントエンドを更新
5. `clients.tenant_id` カラムを削除
6. seed データを更新
7. テスト・ビルド確認

---

## 影響範囲

### 影響が大きい箇所
- 認可エンドポイント（テナント×クライアントの検証ロジック）
- クライアント管理 API（CRUD 全般）
- フロントエンドのクライアント関連ページ

### 影響がない箇所
- ユーザー管理（`users.tenant_id` は変更なし）
- セッション管理
- 署名鍵管理
- JWT 生成・検証ロジック
- Discovery / JWKS エンドポイント

---

## リスクと対策

| リスク | 対策 |
|-------|-----|
| 既存の OIDC フローが壊れる | 移行前に E2E テスト（Phase 3 の RP）で動作を確認し、移行後に再確認 |
| データ移行漏れ | up/down マイグレーションをペアで作成し、ロールバック可能にする |
| 認可時のテナント特定が曖昧になる | 認可エンドポイントの URL に `tenant_code` が含まれるため、テナント特定は既存の仕組みを維持 |
