# OP Frontend - ルール

## 技術スタック

- Next.js 16 (App Router) / React 19 / TypeScript / pnpm
- `output: "export"` — 静的出力（本番は nginx 配信）

## 制約

- 静的エクスポートのため Server Components / API Routes / SSR は使えない
- データ取得は OP Backend API をクライアントサイドで呼び出す
- 環境変数は `NEXT_PUBLIC_` プレフィックス付きのみ使用可能

## ディレクトリ構成

```
src/app/          — App Router のページ
src/components/   — UI コンポーネント
src/lib/          — ユーティリティ
```

## パス別名

- `@/*` → `./src/*`

## 開発コマンド

```bash
docker compose --profile op up -d
# http://localhost:3000
```
