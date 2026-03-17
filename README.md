# OIDC Demo

OIDC（OpenID Connect）準拠の認証基盤（OP: OpenID Provider）と動作検証用RP（Relying Party）の実装プロジェクト。

## 構成

| コンポーネント | 説明 |
|---------------|------|
| `op/backend/` | OIDC認証基盤（Go / Echo / GORM / PostgreSQL） |
| `op/frontend/` | OP管理UI（Next.js、静的出力） |
| `rp/` | 動作検証用RP（Next.js / Drizzle ORM） |

## 環境構築

### .envを作成
```bash
cp .env.example .env
```

### 起動方法

Docker Compose の profiles で起動対象を切り替える。

```bash
# OP のみ起動
docker compose --profile op up -d

# RP のみ起動
docker compose --profile rp up -d

# 全サービス起動
docker compose --profile all up -d
```

### DB管理

**OP Backend** のマイグレーションはサーバー起動時に自動実行される。手動実行も可能。

```bash
# OP マイグレーション手動実行
docker compose exec op-backend go run cmd/migrate/main.go

# 開発用シードデータ投入
docker compose exec op-backend go run cmd/seed/main.go
```

**RP** のマイグレーション（Drizzle）は `rp-migrate` サービスが起動時に自動実行する。手動実行も可能。

```bash
# RP マイグレーション手動実行
docker compose exec rp pnpm db:migrate
```

## ポート

| サービス | ポート |
|---------|--------|
| PostgreSQL | 5432 |
| OP Backend | 8080 |
| OP Frontend | 3000 |
| RP | 3001 |

## ドキュメント

- [設計ドキュメント](docs/design/)
- [実装計画](docs/plan/)
