# OIDC Demo - プロジェクトルール

## プロジェクト概要

OIDC準拠の認証基盤（OP）と動作検証用RP のモノレポ。学習・検証目的。

## 環境変数

- 環境変数は `.env` ファイルで管理する（`.gitignore` 済み）
- `.env.example` に全変数のテンプレートを維持する
- docker-compose.yml に直接シークレットをハードコードしない（`${VAR:-default}` で `.env` から読み込む）
- 新しい環境変数を追加したら `.env.example` も必ず更新する

## ディレクトリ構成

```
op/backend/    — Go (Echo v4 / GORM / PostgreSQL)
op/frontend/   — Next.js 16 (静的出力, pnpm)
rp/            — Next.js 16 (動的, pnpm)
docker/        — PostgreSQL 設定・初期化スクリプト
docs/design/   — 設計ドキュメント
docs/plan/     — 実装計画
```

## Docker Compose

- profiles で起動対象を切り替え: `op`, `rp`, `all`
- `docker compose --profile op up -d` のように使う

## セキュリティ方針

- パスワードハッシュ: argon2id（OWASP推奨パラメータ）
- 署名鍵: AES-256-GCM で暗号化して DB 保存
- セッション: HttpOnly / SameSite=Lax Cookie
- PKCE: S256 のみ（plain は不可）
- Refresh Token: Rotation + Reuse Detection (RFC 9700)
- シークレットをソースコードにハードコードしない

## マイグレーション

- golang-migrate 形式: `NNNNNN_description.{up,down}.sql`
- up と down は必ずペアで作成する
- seed データとスキーマ変更は別ファイルに分離する

## ドキュメント

- 設計に関する内容は `docs/design/` に配置する（README.md に書かない）
- 実装計画は `docs/plan/` に配置する
- README.md はプロジェクト概要・起動方法のみ

## テスト

- Go: テーブル駆動テスト (`func Test_xxx(t *testing.T)`)
- テストファイルは対象と同じパッケージに `_test.go` で配置

## コミット

- 日本語 OK
- 簡潔に変更意図を書く
