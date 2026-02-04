# RP - ルール

## 技術スタック

- Next.js 16 (App Router) / React 19 / TypeScript / pnpm
- openid-client (OIDC クライアント)
- Drizzle ORM / PostgreSQL

## 役割

OP の動作検証用 Relying Party。OIDC フローの受け側を実装する。

## ディレクトリ構成

```
src/app/              — App Router のページ
src/app/api/auth/     — OIDC Callback・Logout (Route Handlers, Node.js Runtime)
src/lib/db/           — Drizzle スキーマ定義
```

## 制約

- OIDC Callback は Route Handlers で実装する（Node.js Runtime 固定）
- DB スキーマは `rp` スキーマを使用（`op` スキーマとは分離）

## パス別名

- `@/*` → `./src/*`

## 開発コマンド

```bash
docker compose --profile rp up -d
# http://localhost:3001
```
