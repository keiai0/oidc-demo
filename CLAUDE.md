# OIDC Demo - プロジェクトルール

## プロジェクト概要

OIDC準拠の認証基盤（OP）と動作検証用RP のモノレポ。学習・検証目的。

## コア原則

1. **シンプル性** — 必要十分な実装に留める。将来の仮定に基づく抽象化や過剰設計をしない
2. **RFC 準拠** — OIDC Core / OAuth 2.1 / RFC 9700 等の仕様を正とし、独自解釈をしない
3. **最小限の影響** — 変更は影響範囲を最小化する。根本原因を特定し、一時的な回避策で済ませない

## ディレクトリ構成

```
op/backend/    — Go (Echo v4 / GORM / PostgreSQL)
op/frontend/   — Next.js 16 (静的出力, pnpm)
rp/            — Next.js 16 (動的, pnpm)
docker/        — PostgreSQL 設定・初期化スクリプト
docs/design/   — 設計ドキュメント
docs/plan/     — 実装計画
docs/research/ — 参考プロジェクト調査（Hydra・Dex）
```

設計の背景・アーキテクチャ方針・パッケージ構成の詳細は `docs/design/04-tech-stack.md` を参照。
参考プロジェクトから採用すべきパターンは `docs/design/05-reference-patterns.md` を参照。

## ワークフロー

### 実装前のドキュメント参照

実装に着手する前に、**そのタスクに関連する**ドキュメントを読む（全部ではなく該当するもののみ）：

- アーキテクチャ・パッケージ構成に関わる → `docs/design/04-tech-stack.md`
- DB スキーマ・エンティティに関わる → `docs/design/02-domain-model.md`
- API エンドポイントの追加・変更 → `docs/design/03-api-design.md`
- 実装スコープの確認 → `docs/plan/` 配下の該当フェーズ計画
- OIDC/OAuth の仕様判断 → RFC 原文（RFC 6749, OpenID Connect Core 1.0, RFC 9700 等）

### Plan モードの活用

以下の条件に該当するタスクは **必ず Plan モードで開始**する：

- RFC / OIDC 仕様の解釈を伴う変更
- 複数パッケージにまたがる変更
- 新しいエンドポイントやフローの追加
- アーキテクチャ・依存関係に影響する変更

Plan モード中に問題が見つかった場合は**無理に進めず立ち止まり、再計画する**。

### 問題発生時の行動指針

- エラーやテスト失敗が発生したら、根本原因を特定してから修正する
- 一時的な回避策（`--no-verify`、エラー握りつぶし等）で突破しない
- 原因が不明な場合はユーザーに相談する（推測で進めない）

## 品質基準

- OIDC エンドポイントのエラーレスポンスは RFC 6749 Section 5.2 / OIDC Core Section 3.1.2.6 に準拠する
- セキュリティに関わるコード（暗号、トークン発行、セッション管理）は OWASP ガイドラインを参照する
- 「経験豊富なセキュリティエンジニアがレビューして承認するか」を判断基準にする
- 仕様に対する疑問が生じたら、実装前に RFC 原文を確認する

## セキュリティ方針

- パスワードハッシュ: argon2id（OWASP推奨パラメータ）
- 署名鍵: AES-256-GCM で暗号化して DB 保存
- セッション: HttpOnly / SameSite=Lax Cookie
- PKCE: S256 のみ（plain は不可）
- Refresh Token: Rotation + Reuse Detection (RFC 9700)
- シークレットをソースコードにハードコードしない

## 環境変数

- `.env` ファイルで管理する（`.gitignore` 済み）
- 新しい環境変数を追加したら `.env.example` も必ず更新する
- docker-compose.yml に直接シークレットをハードコードしない

## Docker Compose

- profiles で起動対象を切り替え: `op`, `rp`, `all`
- `docker compose --profile op up -d` のように使う

## マイグレーション・シード

- DDL: `op/backend/db/migrations/` — golang-migrate 形式
- seed: `op/backend/db/seeds/` — 開発用テストデータ（マイグレーション管理外）
- up と down は必ずペアで作成する
- 1テーブル1ファイル
- `COMMENT ON TABLE` / `COMMENT ON COLUMN` でメタデータコメントを付ける

## Go コーディングルール

### 依存ルール

- **model は何にも依存しない**（エンティティ・DTO の定義のみ）
- **oidc / auth**: deps.go で定義したインターフェース経由で store / crypto / jwt を利用
- **crypto / jwt / store**: model のみに依存（相互依存しない）

### インターフェース設計

- **「Accept interfaces, return structs」** — インターフェースは使う側が `deps.go` に定義する
- store / crypto / jwt は concrete 型を返す
- **純粋関数への依存は関数型で注入する**（例: `VerifyPasswordFunc`, `ComputeATHashFunc`）

### DI

- 全ての組み立ては `cmd/server/main.go` で行う（コンストラクタインジェクション）

### ファイル分割

- 1ファイル1責務を徹底する
- model: 1エンティティ1ファイル
- oidc: 1フロー1ファイル
- store: 1エンティティ1ファイル

### DTO の配置

- 複数パッケージで共有 → `model/`
- フロー固有 → 各パッケージ内

## テスト

- Go: テーブル駆動テスト (`func Test_xxx(t *testing.T)`)
- テストファイルは対象と同じパッケージに `_test.go` で配置

### 検証コマンド

```bash
# Go テスト
cd op/backend && go test ./...

# Go ビルド確認
cd op/backend && go build ./...

# OP Frontend ビルド
cd op/frontend && pnpm build

# RP ビルド
cd rp && pnpm build

# Docker Compose でのビルド・起動
docker compose --profile op up --build -d

# seed 実行
docker compose exec op-backend go run cmd/seed/main.go
```

実装後は**テストとビルドが通ることを確認してから完了**とする。自己申告で完了としない。

## ドキュメント

- 設計に関する内容は `docs/design/` に配置する（README.md に書かない）
- 実装計画は `docs/plan/` に配置する
- README.md はプロジェクト概要・起動方法のみ

## コミット

- 日本語 OK
- 簡潔に変更意図を書く

## よくあるミス・注意事項

- **lestrrat-go/jwx は v3 を使用する**（v2 とは API が異なる）
  - `jwk.Import()`（`jwk.FromRaw()` ではない）
  - `jws.NewHeaders()` + `jws.WithProtectedHeaders()`
  - `token.Get(name, &dst)` で値を取得
- **Echo は v4 を使用する**（v5 は不採用）
- **pnpm を使用する**（npm / yarn ではない）
- **PostgreSQL スキーマ**: OP 用は `op` スキーマ（`op_user` ロール）、RP 用は `rp` スキーマ（`rp_user` ロール）

## 参考プロジェクトからの指針

Hydra・Dex 調査（`docs/research/`）から抽出した、今後の実装で採用すべきパターン（詳細は `docs/design/05-reference-patterns.md`）:

- **Conformance Test**（Dex）: store の各リポジトリに共通テストスイートを導入する
- **コンパイル時インターフェースチェック**（Dex）: `var _ Interface = (*Impl)(nil)` を store に追加する
- **構造化 OIDC エラー型**（Hydra）: Phase 2 で `OIDCError` 型 + Fluent API を導入する
- **time 関数の注入**（Dex）: テスト用に `time.Now()` を差し替え可能にする
- **slog への移行**（Dex）: 構造化ロギングを標準ライブラリで統一する

## 教訓の蓄積

- 実装中に発見した落とし穴やハマりポイントは、このセクションに追記する
- 同じミスを繰り返さないための具体的な対策を記録する
- 修正を受けたら「なぜ間違えたか」をパターンとして残す
