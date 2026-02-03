# OP Backend - Go コーディングルール

## 技術スタック

- Go 1.25 / Echo v4 / GORM v2 / PostgreSQL 17
- lestrrat-go/jwx v3 / golang-migrate
- Air (ホットリロード)

## アーキテクチャ (クリーンアーキテクチャ)

```
handler/       → HTTP の入出力のみ。ビジネスロジックを書かない
usecase/       → アプリケーションロジック。port/ の interface に依存する
domain/        → エンティティ + GORM タグ（実装のシンプルさ優先）
port/          → repository/ と service/ の interface 定義のみ
infrastructure/ → port/ の実装（GORM, jwx, argon2id 等）
```

- handler は usecase または port/service に依存する。infrastructure を直接参照しない
- usecase は port/ の interface にのみ依存する
- infrastructure は port/ の interface を実装する

## コーディング規約

### エラーハンドリング
- エラーは `fmt.Errorf("context: %w", err)` でラップする
- usecase 層のエラーは sentinel error として定義する（例: `usecase/token/errors.go`）
- handler 層では OIDC 仕様に従ったエラーレスポンスを返す

### 命名
- ハンドラ: `XxxHandler` struct + `Handle(c echo.Context) error` メソッド
- リポジトリ: `XxxRepository` interface (port) + 小文字 struct (infrastructure)
- コンストラクタ: `NewXxx(deps...) *Xxx` or `NewXxx(deps...) XxxInterface`

### GORM
- `db.WithContext(ctx)` を必ず使う
- レコード未発見は `(nil, nil)` を返す（`gorm.ErrRecordNotFound` をチェック）

### テスト
- テーブル駆動テスト
- ファイル名: `xxx_test.go`（対象と同じパッケージ）

## lestrrat-go/jwx v3 API

- 鍵インポート: `jwk.Import(rawKey)` （`jwk.FromRaw()` ではない）
- JWS ヘッダ: `jws.NewHeaders()` + `jws.WithProtectedHeaders()`
- トークンクレーム取得: `token.Get(name, &dst)`

## 開発コマンド

```bash
# コンテナ起動
docker compose --profile op up -d

# ログ確認
docker compose logs -f op-backend

# go mod tidy (コンテナ内)
docker compose exec op-backend go mod tidy
```
