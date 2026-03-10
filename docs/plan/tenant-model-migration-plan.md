# テナントモデル移行計画: モデル1 → モデル2

> **ステータス: 実装完了・動作確認済み**

## 概要

テナントモデル（Client が Tenant に 1:n で所属）を、Client と Tenant（Organization）が n:n で関連するモデルに移行する。

## 前提条件

- OIDC コアフロー（Phase 1）が一通り動作していること
- Phase 2（管理機能）が完成していること
- Phase 3（RP動作検証）が完成していること

これらが完了した後に本計画を実施する。

---

## 移行前のモデル（モデル1）

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

### 1. clients テーブルから tenant_id を除去

`000002_create_clients.up.sql` を直接修正し、`tenant_id` カラム・インデックス・コメントを除去した。
運用中のシステムではないため、段階的なマイグレーション（ALTER TABLE）ではなく既存マイグレーションの直接修正で対応。

### 2. 中間テーブルの追加

`000017_create_tenant_clients.up.sql` で `tenant_clients` テーブルを作成。

```sql
CREATE TABLE IF NOT EXISTS tenant_clients (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    client_id   UUID NOT NULL REFERENCES clients(id),
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, client_id)
);
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
| `store/client_repository.go` | `ListByTenantID` → 中間テーブル JOIN に変更、`List`（全件取得）を追加 |
| `store/tenant_client_repository.go` | 新規: `ExistsByTenantAndClient`, `Create`, `Delete`, `ListByClientID` |

### OIDC フロー

| 処理 | 変更内容 |
|-----|---------|
| `oidc/deps.go` | `TenantClientChecker` インターフェース追加 |
| `oidc/authorize.go` | `client.TenantID != tenant.ID` → `tenantClientChecker.ExistsByTenantAndClient()` に変更 |

トークンエンドポイント (`token_authcode.go`, `token_refresh.go`) は変更不要。
テナント情報は `authCode.Session.TenantID` 経由で取得しており、`client.TenantID` を参照していない。

### Management API

| エンドポイント | 変更内容 |
|--------------|---------|
| `GET /management/v1/clients` | 新規: 全クライアント一覧（テナント横断） |
| `GET /management/v1/tenants/:id/clients` | 中間テーブル JOIN で取得 |
| `POST /management/v1/tenants/:id/clients` | クライアント作成 + 中間テーブルへの関連追加 |
| `GET /management/v1/clients/:id/tenants` | 新規: クライアントの関連テナント一覧 |
| `POST /management/v1/clients/:id/tenants` | 新規: クライアントにテナントを紐づけ |
| `DELETE /management/v1/clients/:id/tenants/:tenant_id` | 新規: クライアントからテナント紐づけを解除 |

`management/client.go` の `clientResponse` から `tenant_id` フィールドを削除。
関連テナントは別エンドポイント (`GET /clients/:id/tenants`) で取得する設計。

### deps.go インターフェース

| パッケージ | 変更内容 |
|-----------|---------|
| `oidc/deps.go` | `TenantClientChecker` インターフェース追加 |
| `management/deps.go` | `ClientStore` に `List` 追加、`TenantClientStore` インターフェース追加 |

---

## フロントエンド変更

### 型定義

| ファイル | 変更内容 |
|---------|---------|
| `types/client.ts` | `Client` 型から `tenant_id` を削除、`TenantAssociation` 型を追加 |
| `types/index.ts` | `TenantAssociation` を re-export |

### API 層

| ファイル | 変更内容 |
|---------|---------|
| `lib/api/clients.ts` | `listAll()`, `listTenants()`, `addTenant()`, `removeTenant()` 追加 |
| `lib/query/query-keys.ts` | `clients.list`, `clients.tenants` 追加 |

### ページ変更

| ページ | 変更内容 |
|-------|---------|
| クライアント一覧 (`/management/clients`) | N+1 クエリ → `listAll` 単一 API に置き換え |
| クライアント詳細 (`/management/clients/detail`) | 「関連テナント」セクション追加（追加・解除 UI）、削除後のリダイレクト先をグローバルクライアント一覧に変更 |

---

## Seed データ

`db/seeds/seed.sql` の `clients` INSERT から `tenant_id` を除去し、`tenant_clients` INSERT を追加。

---

## 開発ツール

マイグレーションの直接修正に伴い、DB 再作成を容易にする `fresh` コマンドを追加した。

```bash
# 通常のマイグレーション（差分適用）
go run cmd/migrate/main.go

# 全テーブル削除 → マイグレーション再実行 → seed 自動投入
go run cmd/migrate/main.go fresh
```

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
