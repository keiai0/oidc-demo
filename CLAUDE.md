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

## マイグレーション・シード

- DDL: `op/backend/db/migrations/` — golang-migrate 形式、テーブルごとに分割
- seed: `op/backend/db/seeds/` — 開発用テストデータ（マイグレーション管理外）
- マイグレーションはサーバー起動時に自動実行
- seed は `docker compose exec op-backend go run cmd/seed/main.go` で手動実行
- up と down は必ずペアで作成する
- `COMMENT ON TABLE` / `COMMENT ON COLUMN` でメタデータコメントを付ける

## ドキュメント

- 設計に関する内容は `docs/design/` に配置する（README.md に書かない）
- 実装計画は `docs/plan/` に配置する
- README.md はプロジェクト概要・起動方法のみ

## テスト

- Go: テーブル駆動テスト (`func Test_xxx(t *testing.T)`)
- テストファイルは対象と同じパッケージに `_test.go` で配置

## アーキテクチャ（Go × フロー中心アーキテクチャ）

認証基盤では「ビジネスロジック」の大半が RFC/仕様で決まっており、usecase 層に書くことが仕様手順の写しになりがち。
OIDC の各フロー（authorize, token, userinfo 等）をそのままパッケージ・ファイル単位とし、
handler とロジックを同居させるフロー中心の構成を採用する。

### パッケージ構成

```
op/backend/internal/
├── model/            — エンティティ・DTO（依存なし・最内層）
├── oidc/             — OIDC フロー（HTTP ハンドラ + ロジック一体）
│   ├── authorize.go          — 認可エンドポイント
│   ├── token.go              — トークンエンドポイント（リクエスト解析・ルーティング）
│   ├── token_authcode.go     — Authorization Code Grant フロー
│   ├── token_refresh.go      — Refresh Token Grant フロー
│   ├── userinfo.go           — UserInfo エンドポイント
│   ├── revoke.go             — Revocation エンドポイント
│   ├── discovery.go          — Discovery エンドポイント
│   ├── jwks.go               — JWKS エンドポイント
│   ├── deps.go               — 依存インターフェース・関数型
│   └── errors.go             — OIDC エラーヘルパー
├── auth/             — 内部認証 API（OP Frontend 向け）
│   ├── login.go              — ログイン
│   ├── me.go                 — ユーザー情報取得
│   ├── deps.go
│   └── errors.go
├── crypto/           — 暗号処理（argon2id, AES, PKCE）
├── jwt/              — JWT 署名・検証・鍵管理
├── store/            — 永続化（GORM 実装）
└── database/         — DB 接続・マイグレーション
```

### 依存ルール

- **model は何にも依存しない**（エンティティ・DTO の定義のみ）
- **oidc / auth**: model, crypto, jwt, store のインターフェースに依存（deps.go で定義）
- **crypto / jwt / store**: model のみに依存（相互依存しない）
- **パッケージ間の import は自然な方向で許可**（クリーンアーキテクチャの層制約は設けない）

### インターフェース設計（Go 慣習）

- **「Accept interfaces, return structs」** — インターフェースは使う側が定義する
- `port/` のような共有インターフェースディレクトリは使わない
- 各パッケージが必要とするインターフェースをパッケージ内の `deps.go` に定義する
- store / crypto / jwt は concrete 型（exported struct / 関数）を返す
- Go の暗黙的インターフェース実装により、import なしで依存が成立する

### DI（依存性注入）

- 全ての組み立ては `cmd/server/main.go` で行う
- コンストラクタインジェクションを使用
- oidc・auth パッケージはローカル定義のインターフェースを受け取り、具象型を知らない
- **純粋関数への依存は関数型で注入する**（例: `VerifyPasswordFunc`, `ComputeATHashFunc`）
  - ステートレスな暗号処理等にインターフェースは不要

### DTO の配置

- 複数パッケージで共有される DTO（`LoginInput`, `IDTokenClaims` 等）は `model/` に配置する
- フロー固有の入出力型（`AuthCodeGrantInput`, `TokenResponse` 等）は各パッケージ内に定義する

## ファイル分割

- 1ファイル1責務を徹底する
- 「似ている」「関連がある」だけの理由で複数の型・インターフェースを1ファイルにまとめない
- model: 1エンティティ1ファイル（例: `client.go`, `redirect_uri.go`）
- deps.go: パッケージ内で共有するインターフェース・関数型をまとめる
- oidc: 1フロー1ファイル（例: `authorize.go`, `token_authcode.go`）
- store: 1エンティティ1ファイル
- マイグレーション: 1テーブル1ファイル

## コミット

- 日本語 OK
- 簡潔に変更意図を書く
