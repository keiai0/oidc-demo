# Dex リポジトリ徹底分析レポート

> **対象リポジトリ**: https://github.com/dexidp/dex
> **調査日**: 2026-02-27
> **調査対象バージョン**: v2.45.0 (コミット 47e84db, 2026-02-26)
> **目的**: Go製OAuth2/OIDC機能の新規開発に向けた設計・実装パターンの調査

---

## Part 1: プロジェクト概要

> Dexは「Federated OpenID Connect Provider」— 複数のIDプロバイダ（GitHub, Google, LDAP, SAML等）とアプリケーションの間に立つ認証ブリッジである。

### プロジェクトの目的・スコープ

- **目的**: OAuth2/OIDC仕様に準拠したIDトークン発行サーバーを提供し、アプリケーションが個別のIDプロバイダ実装を持たずに済むようにする
- **分類**: **アプリケーション**（ライブラリでもフレームワークでもない）。単体デプロイ可能なOIDCサーバー
- **解決課題**: 異なるIDプロバイダ間のプロトコル差異を吸収し、統一されたOIDCインターフェースで認証を提供
- **主要ユースケース**: Kubernetes認証、Argo CD SSO、sigstoreのFulcio等

### 設計哲学

1. **標準準拠**: OAuth2/OIDC仕様を忠実に実装（RFC 6749, RFC 7636 PKCE, RFC 8628 Device Flow, RFC 7662 Token Introspection）
2. **プラグイン指向**: Connector（IDプロバイダ）とStorage（永続化）をインターフェースで抽象化し差し替え可能に
3. **Kubernetesネイティブ**: CRDベースのストレージをサポートし、K8sクラスタ内で外部DBなしに動作可能
4. **設定駆動**: YAMLファイルによるデプロイ設定。gRPC APIで動的変更も可能

### Goバージョン・主要依存ライブラリ

| カテゴリ | ライブラリ | 用途 |
|---------|----------|------|
| **Go** | **1.25.0** | - |
| OIDC | `coreos/go-oidc/v3` | OIDCクライアント実装 |
| JWT | `go-jose/go-jose/v4` | JWT生成・検証 |
| HTTP | `gorilla/mux` | ルーティング |
| HTTP | `gorilla/handlers` | CORS等ミドルウェア |
| gRPC | `google.golang.org/grpc` v1.79.1 | 管理API |
| CLI | `spf13/cobra` | コマンドライン |
| ORM | `entgo.io/ent` v0.14.5 | DB操作（実験的） |
| 監視 | `prometheus/client_golang` | メトリクス |
| ヘルスチェック | `AppsFlyer/go-sundheit` | ヘルスチェック |
| テスト | `stretchr/testify` | アサーション |
| ログ | `log/slog`（標準ライブラリ） | 構造化ログ |
| LDAP | `go-ldap/ldap/v3` | LDAPコネクタ |
| SAML | `russellhaering/goxmldsig` | SAML署名検証 |
| Vault | `openbao/openbao/api/v2` | 署名鍵管理 |
| DB | `lib/pq`, `go-sql-driver/mysql`, `mattn/go-sqlite3` | SQL |
| KVS | `go.etcd.io/etcd/client/v3` | etcd |
| 暗号 | `golang.org/x/crypto` | bcrypt等 |

### プロジェクトの成熟度

| 指標 | 値 |
|------|-----|
| 最終コミット日 | 2026-02-26（調査前日） |
| 最新タグ | v2.45.0 |
| 主要コントリビューター | Maksim Nabokikh (21), Márk Sági-Kazár (34), dependabot (83) |
| ライセンス | Apache License 2.0 |
| CNCFステータス | CNCFサンドボックスプロジェクト |
| Go総行数 | 約85,000行（本番68,700行 + テスト16,300行） |
| Goファイル数 | 246ファイル（テスト46ファイル） |
| 主要採用企業 | Argo CD, Kubeflow, sigstore, Chef, Kyma他 |

---

## Part 2: アーキテクチャ分析

### ディレクトリ構成

```
dex/
├── api/                    # gRPC API定義（Protocol Buffers）
│   ├── v2/                 # API v2（現行バージョン）
│   ├── api.pb.go           # 生成コード
│   └── api_grpc.pb.go      # gRPCサービス定義
├── cmd/                    # CLIエントリーポイント
│   ├── dex/                # メインサーバーバイナリ（7ファイル）
│   └── docker-entrypoint/  # Docker用エントリーポイント
├── connector/              # IDプロバイダ連携（17種）
│   ├── connector.go        # コアインターフェース定義
│   ├── github/             # GitHub OAuth2
│   ├── oidc/               # 汎用OIDC
│   ├── ldap/               # LDAP認証
│   ├── saml/               # SAML 2.0
│   ├── google/             # Google
│   ├── microsoft/          # Azure AD
│   └── mock/               # テスト用
├── server/                 # OIDCサーバーコア（約40ファイル）
│   ├── server.go           # Server構造体と初期化
│   ├── handlers.go         # HTTPハンドラー
│   ├── oauth2.go           # OAuth2フロー実装
│   ├── refreshhandlers.go  # リフレッシュトークン処理
│   ├── signer/             # トークン署名（Local, Vault）
│   └── internal/           # 内部型（protobuf）
├── storage/                # 永続化レイヤー
│   ├── storage.go          # Storageインターフェース（460行）
│   ├── static.go           # 静的設定デコレータ
│   ├── conformance/        # ストレージ適合テスト
│   ├── memory/             # インメモリ実装
│   ├── sql/                # SQL実装（PostgreSQL, MySQL, SQLite）
│   ├── ent/                # Ent ORM実装（実験的）
│   ├── kubernetes/         # Kubernetes CRD実装
│   └── etcd/               # etcd実装
├── pkg/                    # ユーティリティ
│   ├── featureflags/       # フィーチャーフラグ
│   ├── groups/             # グループフィルタリング
│   └── httpclient/         # HTTPクライアントファクトリ
├── web/                    # フロントエンドアセット
├── examples/               # サンプルアプリケーション
├── docs/                   # ドキュメント
└── scripts/                # ビルド・デプロイスクリプト
```

### パッケージ間の依存方向

```
cmd/dex（エントリーポイント）
  ↓ 設定解析・初期化
server（OIDCサーバーコア）
  ├──→ connector（認証委譲）
  ├──→ storage（データ永続化）
  ├──→ server/signer（トークン署名）
  └──→ pkg/（ユーティリティ）
        ↑
connector → pkg/httpclient（HTTP通信）
server/signer → storage（署名鍵の永続化）
```

**評価**: 依存方向は概ね一方向で整理されている。`server` パッケージがハブとなり、`connector` と `storage` が独立して差し替え可能な設計。ただし `server` パッケージ自体は全コネクタの具体型をインポートしておりやや結合が強い（`server/server.go:31-49`）。

### 設計パターン

#### 1. Strategy パターン（Connector / Storage / Signer）

3つの主要インターフェースでStrategy パターンを採用。実行時に実装を差し替え可能。

```go
// connector/connector.go - 認証戦略
type CallbackConnector interface {
    LoginURL(s Scopes, callbackURL, state string) (string, []byte, error)
    HandleCallback(s Scopes, connData []byte, r *http.Request) (Identity, error)
}

// storage/storage.go - 永続化戦略
type Storage interface {
    CreateAuthRequest(ctx context.Context, a AuthRequest) error
    GetAuthRequest(ctx context.Context, id string) (AuthRequest, error)
    // ... 39メソッド
}

// server/signer/signer.go - 署名戦略
type Signer interface {
    Sign(ctx context.Context, payload []byte) (string, error)
    ValidationKeys(ctx context.Context) ([]*jose.JSONWebKey, error)
    Algorithm(ctx context.Context) (jose.SignatureAlgorithm, error)
    Start(ctx context.Context)
}
```

#### 2. Factory パターン（Connector/Storage登録）

文字列名からConfig構造体を生成するファクトリマップ:

```go
// server/server.go:676-696
var ConnectorsConfig = map[string]func() ConnectorConfig{
    "oidc":   func() ConnectorConfig { return new(oidc.Config) },
    "github": func() ConnectorConfig { return new(github.Config) },
    // ...
}

// cmd/dex/config.go:330-337
var storages = map[string]func() StorageConfig{
    "postgres":   getORMBasedSQLStorage(...),
    "kubernetes": func() StorageConfig { return new(kubernetes.Config) },
    // ...
}
```

#### 3. Decorator パターン（Static Storage Wrappers）

`storage/static.go` で静的設定をオーバーレイするデコレータを実装:

```go
// storage/static.go
type staticClientsStorage struct {
    Storage                         // インターフェース埋め込み
    clients     []Client
    clientsByID map[string]Client
}

func WithStaticClients(s Storage, staticClients []Client) Storage
func WithStaticPasswords(s Storage, staticPasswords []Password, logger *slog.Logger) Storage
func WithStaticConnectors(s Storage, staticConnectors []Connector) Storage
```

#### 4. Compare-And-Swap（楽観的並行制御）

Storageの更新メソッドはupdater関数パターンでCASセマンティクスを実現:

```go
// storage/storage.go:131-134
UpdateClient(ctx context.Context, id string,
    updater func(old Client) (Client, error)) error
UpdateRefreshToken(ctx context.Context, id string,
    updater func(r RefreshToken) (RefreshToken, error)) error
```

#### 5. Embedded Interface（インターフェース埋め込みによるデコレータ）

```go
// server/server.go:626-631 - キーキャッシュ
type keyCacher struct {
    storage.Storage           // Storage全メソッドを委譲
    now  func() time.Time
    keys atomic.Value         // GetKeysだけオーバーライド
}
```

#### インターフェース設計の方針

- **小さなインターフェース**: Connectorは用途別に4つの小インターフェースに分割（PasswordConnector 2メソッド、CallbackConnector 2メソッド、SAMLConnector 2メソッド、RefreshConnector 1メソッド）
- **大きなStorage**: 対照的にStorageインターフェースは39メソッドの大きなインターフェース（後述の注意点）
- **マーカーインターフェース**: `Connector interface{}` は空で、型アサーションで具体型を判別

#### 構造体の設計

- **Config構造体による依存注入**: `server.Config` に全依存を集約（`server/server.go:63-128`）
- **Functional Optionsは不使用**: 代わりにConfig構造体のゼロ値にデフォルトを適用するパターン
- **time関数の注入**: `Now func() time.Time` をConfig経由で渡しテスタビリティを確保

#### エラーハンドリング

- **センチネルエラー**: `storage.ErrNotFound`, `storage.ErrAlreadyExists`（`storage/storage.go:17-22`）
- **カスタムエラー型**: OAuth2エラーレスポンス用の `displayedAuthErr`, `redirectedAuthErr`（`server/oauth2.go:34-88`）
- **gRPCレスポンス**: エラーコードではなくレスポンス構造体のフラグで返す（`api/v2`）

#### context.Context の活用

- 全Storage/Connectorメソッドの第一引数に`context.Context`
- リクエストIDとリモートIPをcontextに格納（`server/server.go:789-795`）
- `slog.Logger` の `ErrorContext`/`InfoContext` でcontext値をログ出力
- goroutineのキャンセル制御に使用（GC、鍵ローテーション）

---

## Part 3: Go実装の詳細分析

### パッケージ構成の評価

**事実**: Go公式の推奨レイアウト（`cmd/`, `internal/`, `pkg/`）に概ね準拠。

- `cmd/dex/` にメインパッケージ: **標準的**
- `pkg/` は3パッケージのみ: 控えめで良い
- `internal/` は `server/internal/` に1つだけ: protobuf型のみ

**評価（主観）**: パッケージ構成はフラットで分かりやすい。トップレベルに `connector/`, `storage/`, `server/` とドメイン概念がそのまま表現されている。Go Standard Project Layoutの `/internal/` をもっと活用できる余地はある。

### internal パッケージの活用

**事実**: `server/internal/` のみ。protobufで生成された内部型（`types.pb.go`）と`codec.go`の2ファイル。

**評価（主観）**: 活用が非常に少ない。コネクタの具体実装やストレージの内部型など、外部に公開すべきでないものが `internal` なしに公開されている。新規プロジェクトではより積極的に使うべき。

### エクスポート/非エクスポートの判断

- コネクタ実装の構造体は非エクスポート（例: `oidcConnector`）、Configはエクスポート
- `memStorage` は非エクスポート、ファクトリ関数 `New()` のみエクスポート
- サーバーの内部ハンドラーメソッドは全て非エクスポート（`handleToken`, `handleAuthorization`等）

### 命名規則

| 要素 | 規則 | 例 |
|------|------|-----|
| パッケージ名 | 短い小文字 | `storage`, `connector`, `signer` |
| インターフェース名 | 役割を表す名詞 | `Storage`, `Signer`, `CallbackConnector` |
| レシーバ名 | 1-2文字 | `s *Server`, `k *keyCacher`, `db passwordDB` |
| 変数名 | 短縮形多用 | `ctx`, `req`, `conn`, `c` |
| エラー変数 | `Err` プレフィックス | `ErrNotFound`, `ErrAlreadyExists` |
| ファクトリ関数 | `New` プレフィックス | `NewServer`, `NewAPI`, `New` |
| テストヘルパー | `new` / `must` プレフィックス | `newTestServer`, `mustLoadJWK` |

**評価（主観）**: Go標準の命名規約にほぼ準拠。Effective Goの推奨通り。

### ファイル分割の粒度

- `server/` パッケージ: 機能単位で分割（`handlers.go`, `oauth2.go`, `refreshhandlers.go`, `deviceflowhandlers.go`）
- 各コネクタ: 1パッケージ=1-2ファイル（本体 + テスト）
- ストレージ: バックエンドごとにサブパッケージ

### GoDocコメント

- パッケージレベルのdoc.goは `server/doc.go` と `storage/doc.go` に存在
- インターフェースメソッドには詳細なコメント（特にStorage, Connectorインターフェース）
- RFC参照リンクがコード内に多数記載

### 並行処理パターン

**goroutine**:
- GCループ: `server/server.go:650-667` — `go func()` + `select` + `time.After` パターン
- 鍵ローテーション: `server/signer/local.go` — 同様のバックグラウンドgoroutine

**sync**:
- `sync.Mutex`: Server.connectors保護（`server/server.go:172`）、メモリストレージ全体（`storage/memory/memory.go:44`）
- `sync/atomic.Value`: キーキャッシュ（`server/server.go:630`）

**channel**: 直接的なchannel使用はほぼない。`context.Done()` と `time.After` で制御。

### go generate の活用

```go
// storage/ent/generate.go
//go:generate go tool entgo.io/ent/cmd/ent generate ./schema --target ./db
```

Ent ORMのスキーマからDBクライアントコードを自動生成。Protocol Buffersの生成はMakefileで管理（`make generate-proto`）。

---

## Part 4: テスト戦略

> テストフレームワークはGo標準の`testing`パッケージをベースに、`testify`でアサーション強化。特筆すべきはストレージの適合テスト（conformance test）パターン。

### テストフレームワーク

| ライブラリ | 用途 |
|-----------|------|
| `testing` | 標準テストフレームワーク |
| `stretchr/testify` | `require.XXX()` / `assert.XXX()` アサーション |
| `net/http/httptest` | HTTPサーバー/レコーダー |
| `kylelemons/godebug/pretty` | テスト出力のpretty print |

### テストの種類と比率

| 種類 | ファイル数 | 概要 |
|------|----------|------|
| ユニットテスト | 約30 | ハンドラー、設定パース、URL検証等 |
| 統合テスト | 約10 | 実DB、LDAP、Vault連携 |
| 適合テスト | 1（共有） | 全ストレージバックエンド共通 |
| E2Eパターン | 数ファイル | httptest.NewServer利用のフルフロー |

テストコード比率: 約19%（16,340行 / 85,043行）

### 適合テスト（Conformance Tests）— 最重要パターン

`storage/conformance/conformance.go` で全ストレージバックエンドに共通のテストスイートを提供:

```go
// storage/conformance/conformance.go:40-55
func RunTests(t *testing.T, newStorage func(t *testing.T) storage.Storage) {
    runTests(t, newStorage, []subTest{
        {"AuthCodeCRUD", testAuthCodeCRUD},
        {"AuthRequestCRUD", testAuthRequestCRUD},
        {"ClientCRUD", testClientCRUD},
        {"RefreshTokenCRUD", testRefreshTokenCRUD},
        {"PasswordCRUD", testPasswordCRUD},
        {"KeysCRUD", testKeysCRUD},
        {"OfflineSessionCRUD", testOfflineSessionCRUD},
        {"ConnectorCRUD", testConnectorCRUD},
        {"GarbageCollection", testGC},
        {"TimezoneSupport", testTimezones},
        {"DeviceRequestCRUD", testDeviceRequestCRUD},
        {"DeviceTokenCRUD", testDeviceTokenCRUD},
    })
}
```

各ストレージ実装のテストファイルでは:
```go
// storage/memory/memory_test.go
func TestStorage(t *testing.T) {
    conformance.RunTests(t, func(t *testing.T) storage.Storage {
        return New(slog.New(...))
    })
}
```

### テーブル駆動テスト

広範に使用。`t.Run()` を含む箇所は78箇所以上:

```go
// server/handlers_test.go 典型パターン
tests := []struct {
    name     string
    field1   type1
    expected type3
}{
    {"case 1", val1, expected1},
    {"case 2", val2, expected2},
}
for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) {
        // テスト実装
    })
}
```

### モック・スタブの作り方

- **手書きモック**: `server/signer/mock.go`（MockSigner）、`connector/mock/`（テスト用コネクタ）
- **gomock等のコード生成は不使用**
- **部分実装パターン**: インターフェース埋め込みで必要メソッドだけオーバーライド

```go
// server/handlers_test.go
type emptyStorage struct { storage.Storage }  // 未実装メソッドはpanic
```

### 統合テストの制御

環境変数でスキップ:
```go
func TestPostgres(t *testing.T) {
    host := os.Getenv("DEX_POSTGRES_HOST")
    if host == "" {
        t.Skipf("test environment variable %s not set, skipping", ...)
    }
}
```

ビルド制約:
```go
//go:build cgo  // SQLiteテスト用
```

### テストファイルの配置

- テストファイルは同一パッケージ内（`_test.go` サフィックス）
- テストデータは `testdata/` ディレクトリ（`connector/ldap/testdata/`, `connector/saml/testdata/` 等）
- 外部テスト用パッケージ（`_test` サフィックス）は使用されていない

---

## Part 5: CI/CD・開発ツール

### CI構成

**GitHub Actions** (`/.github/workflows/`):

| ワークフロー | トリガー | 内容 |
|-------------|---------|------|
| `ci.yaml` | push to master, PR | テスト全体（Postgres, MySQL, etcd, Vault, LDAP, Kubernetes） |
| `artifacts.yaml` | 再利用可能 | マルチアーキDockerイメージビルド、SBOM、cosign署名 |
| `release.yaml` | タグ `v*` | artifacts.yaml呼び出し + GHCR/DockerHub公開 |
| `checks.yaml` | PR | リリースノートラベル検証 |
| `analysis-scorecard.yaml` | スケジュール | OpenSSFセキュリティスコアカード |
| `trivydb-cache.yaml` | スケジュール | Trivy脆弱性DBキャッシュ |

### Linter設定

`.golangci.yaml` （golangci-lint v2.4.0）:

**有効なリンター（13個）**:
- `depguard`, `dogsled`, `exhaustive`, `gochecknoinits`, `goprintffuncname`, `govet`, `ineffassign`, `misspell`, `nakedret`, `nolintlint`, `prealloc`, `unconvert`, `unused`, `whitespace`

**フォーマッタ（4個）**:
- `gci`（import整理）, `gofmt`, `gofumpt`, `goimports`

**注目設定**:
- `depguard`: `io/ioutil` を禁止（非推奨パッケージの検出）
- `staticcheck` と `errcheck` は**無効化**されている
- テストファイルでは `errcheck` と `noctx` を除外

**評価（主観）**: `staticcheck` と `errcheck` の無効化は意外。特に `errcheck` はエラー見落とし防止に重要であり、新規プロジェクトでは有効にすべき。

### Makefile のタスク構成

| ターゲット | 内容 |
|-----------|------|
| `build` | Dexバイナリビルド |
| `test` / `testrace` / `testall` | テスト実行（gotestsum使用） |
| `lint` / `fix` | リンター実行/自動修正 |
| `generate` | proto + ent コード生成 |
| `verify` | 生成コードの差分チェック |
| `up` / `down` | docker-compose開発環境 |
| `kind-up` / `kind-tests` / `kind-down` | Kubernetesテスト |
| `deps` | 開発ツールインストール |

### リリース方法

- セマンティックバージョニング（`v2.45.0`）
- Gitタグによるリリーストリガー
- マルチアーキDockerイメージ（alpine + distroless）
- SBOM（Software Bill of Materials）生成、cosign署名

### コミットメッセージ規約

Conventional Commits風:
```
feat(connector): add compile-time checks for connector interfaces
fix(mysql): quote `groups` reserved word in query replacer
build(deps): bump actions/attest-build-provenance from 3.2.0 to 4.0.0
test: update HandleCallback after merging OIDC PKCE
```

### PR/Issueテンプレート

- **PR**: 概要、変更理由、関連Issue（`Closes #`）、レビュアーへの特記事項
- **Bug Report**: バージョン、ストレージタイプ、インストール方法、再現手順、設定YAML、ログ
- **Feature Request**: 問題説明、提案解決策、代替案、追加コンテキスト
- **DCO（Developer Certificate of Origin）**: コミット署名必須

---

## Part 6: 参考にすべき点（✅ Adopt）

### 1. ストレージ適合テスト（Conformance Test）パターン — 最も学ぶべき設計

**ファイル**: `storage/conformance/conformance.go`

```go
func RunTests(t *testing.T, newStorage func(t *testing.T) storage.Storage) {
    runTests(t, newStorage, []subTest{
        {"AuthCodeCRUD", testAuthCodeCRUD},
        // ... 12テスト
    })
}
```

**なぜ優れているか**: インターフェースの契約をテストとして形式化している。新しいストレージバックエンドを追加する際、`conformance.RunTests()` を呼ぶだけで全CRUDの正しさが検証される。**テスタビリティと拡張性の両方を一度に確保する秀逸なパターン**。

**適用方法**: OAuth2サーバーの新規開発でもStorageインターフェースの適合テストを最初に書き、全バックエンドが一貫した振る舞いを保証する。

### 2. Update関数パターン（楽観的ロック）

**ファイル**: `storage/storage.go:117-138`

```go
UpdateClient(ctx context.Context, id string,
    updater func(old Client) (Client, error)) error
```

**なぜ優れているか**: ストレージ側でトランザクション管理を隠蔽しつつ、呼び出し元はread-modify-writeのロジックだけに集中できる。updater関数が複数回呼ばれる可能性を許容することで、各バックエンドが最適なリトライ戦略を採用できる。

### 3. Connector インターフェースの分離設計

**ファイル**: `connector/connector.go`

```go
type Connector interface{}                    // マーカー
type PasswordConnector interface { ... }      // 2メソッド
type CallbackConnector interface { ... }      // 2メソッド
type RefreshConnector interface { ... }       // 1メソッド
type TokenIdentityConnector interface { ... } // 1メソッド
```

**なぜ優れているか**: Interface Segregation Principleの好例。コネクタが自分の対応する認証フローだけを実装すればよい。サーバー側は型アサーションで対応フローを判別。小さなインターフェースの組み合わせという、Goらしい（idiomatic）設計。

### 4. コンパイル時インターフェースチェック

**ファイル**: 各コネクタ実装、`cmd/dex/config.go:288-298`

```go
var _ StorageConfig = (*etcd.Etcd)(nil)
var _ StorageConfig = (*kubernetes.Config)(nil)
var _ StorageConfig = (*sql.Postgres)(nil)
```

最新コミット（`47e84db`）でConnectorにも適用:
```go
var _ connector.CallbackConnector = (*githubConnector)(nil)
```

**適用方法**: 全インターフェース実装にこのパターンを採用し、コンパイル時に契約違反を検出する。

### 5. time関数の注入によるテスタビリティ

**ファイル**: `server/server.go:112`, `server/signer/rotation.go:53`

```go
type Config struct {
    Now func() time.Time  // nilならtime.Now
}
```

**なぜ優れているか**: 時間依存のロジック（トークン有効期限、鍵ローテーション、GC）を決定論的にテスト可能にする。OAuth2/OIDCは時間に強く依存するドメインなので特に重要。

### 6. Static Storage Decoratorパターン

**ファイル**: `storage/static.go`

設定ファイルで定義された静的クライアント/パスワード/コネクタを、動的ストレージの上にデコレータとして重ねる設計。Get/List操作は静的値を優先し、Create/Update/Deleteは静的エントリにはエラーを返す。

**なぜ優れているか**: 設定ファイルベースの管理とAPI動的管理を同一Storageインターフェースで統一的に扱える。

### 7. セキュリティヘッダーの設定可能化

**ファイル**: `cmd/dex/config.go:212-256`

Content-Security-Policy, X-Frame-Options, Strict-Transport-Security等のセキュリティヘッダーを設定で制御可能。OIDCサーバーとして適切なデフォルトセキュリティ。

### 8. Docker Entrypoint のgomplate活用

**ファイル**: `Dockerfile`, `cmd/docker-entrypoint/`

設定ファイルのテンプレート処理をgomplateで行い、環境変数から動的設定生成。Kubernetesデプロイでの柔軟性を確保。

---

## Part 7: 参考にすべきでない点（⚠️ Avoid）

### 1. 巨大な Storage インターフェース（39メソッド）

**ファイル**: `storage/storage.go:76-143`

**理由**: 39メソッドのインターフェースはGoの「小さなインターフェース」哲学に反する。新しいエンティティを追加するたびに全バックエンド実装を変更する必要がある。Go Code Review Commentsの "The bigger the interface, the weaker the abstraction" に抵触。

**代替案**:
```go
type AuthRequestStore interface {
    CreateAuthRequest(ctx context.Context, a AuthRequest) error
    GetAuthRequest(ctx context.Context, id string) (AuthRequest, error)
    DeleteAuthRequest(ctx context.Context, id string) error
    UpdateAuthRequest(ctx context.Context, id string, updater func(AuthRequest) (AuthRequest, error)) error
}

type ClientStore interface { /* ... */ }
type TokenStore interface { /* ... */ }

// 必要に応じて合成
type Storage interface {
    AuthRequestStore
    ClientStore
    TokenStore
    // ...
    io.Closer
}
```

### 2. コネクタ登録のハードコード

**ファイル**: `server/server.go:676-696`

```go
var ConnectorsConfig = map[string]func() ConnectorConfig{
    "github":  func() ConnectorConfig { return new(github.Config) },
    "oidc":    func() ConnectorConfig { return new(oidc.Config) },
    // ... 全コネクタが列挙
}
```

**理由**: 新しいコネクタを追加するには `server` パッケージを変更する必要がある。`server` パッケージが全コネクタの具体型をインポートしており、不要な依存が生まれる。

**代替案**: `init()` ベースのプラグイン登録か、Goのビルドタグによる選択的ビルド。ただしこれはDexが「単体アプリケーション」であるために選ばれた設計であり、ライブラリとして使う場合に問題となるパターン。新規プロジェクトがアプリケーションならこのパターンも許容できる。

### 3. pkg/errors の使用

**ファイル**: `go.mod` — `github.com/pkg/errors v0.9.1`

**理由**: Go 1.13以降の標準 `errors` パッケージ（`errors.Is`, `errors.As`, `fmt.Errorf("%w",...)`）で十分。`pkg/errors` はメンテナンス終了しており、新規プロジェクトで採用する理由はない。

**代替案**: 標準ライブラリの `errors` + `fmt.Errorf` のみを使用。

### 4. gorilla/mux の使用

**ファイル**: `server/server.go:426`

**理由**: gorilla/muxは2022年にアーカイブされ、後にメンテナンスが再開されたが、Go 1.22以降の標準 `net/http` のルーティング機能（パスパラメータ、メソッドマッチング）で多くのユースケースをカバーできる。

**代替案**: Go 1.22+の `http.NewServeMux()` またはより軽量な `chi` ルーター。

### 5. エラーハンドリングの不統一

**ファイル**: `server/oauth2.go:29` のTODOコメント

```go
// TODO(ericchiang): clean this file up and figure out more idiomatic error handling.
```

また `storage/storage.go:90` にも:
```go
// TODO(ericchiang): return (T, bool, error) so we can indicate not found
// requests that way instead of using ErrNotFound.
```

**理由**: sentinel error (`ErrNotFound`) とカスタムエラー型 (`displayedAuthErr`, `redirectedAuthErr`) と素の `fmt.Errorf` が混在。コード内のTODOコメントが自己認識している通り、統一されていない。

**代替案**: カスタムエラー型 + `errors.Is`/`errors.As` を一貫して使用。エラーの分類体系を最初に設計する。

### 6. errcheck/staticcheck の無効化

**ファイル**: `.golangci.yaml`

```yaml
linters:
  disable:
    - staticcheck
    - errcheck
```

**理由**: `errcheck` はエラーの見落としを防ぐ最も基本的なリンター。`staticcheck` はGoの最も包括的な静的解析ツール。両方の無効化は品質リスク。

**代替案**: 新規プロジェクトでは最初から `errcheck`, `staticcheck`, `govet` を必ず有効にし、必要最小限の除外ルールのみ設定する。

### 7. JSON タグのバグ放置

**ファイル**: `storage/storage.go:390`

```go
type Connector struct {
    // ...
    // NOTE: This is a bug. The JSON tag should be `config`.
    Config []byte `json:"email"`
}
```

**理由**: 後方互換性のためにバグが意図的に残されている。Kubernetesオブジェクトのマイグレーションコストが高いため。

**教訓**: 新規プロジェクトではJSONタグを最初から正しく設計する。特にKubernetes CRDやAPIで使う場合、後から変更するのは極めて困難。

### 8. Genericsの未活用

**ファイル**: `server/api.go:577` — プロジェクト全体で1箇所のみ

```go
func defaultTo[T comparable](v, def T) T {
```

**理由**: Go 1.25なのにgenericsがほぼ使われていない。ストレージのCRUD操作やAPIレスポンス変換など、genericsで型安全に共通化できる箇所が多数ある。歴史的経緯（Go 1.18以前からの設計）。

**代替案**: 新規プロジェクトではCRUD操作、リスト操作、エラーラッピング等にgenericsを積極的に活用:
```go
type Repository[T any] interface {
    Create(ctx context.Context, entity T) error
    Get(ctx context.Context, id string) (T, error)
    Update(ctx context.Context, id string, updater func(T) (T, error)) error
    Delete(ctx context.Context, id string) error
}
```

### 9. slogへの移行が不完全

**事実**: `log/slog` と `github.com/pkg/errors` が混在。一部のコネクタやストレージでは古いログパターンが残っている可能性。

**代替案**: 新規プロジェクトでは `log/slog` を唯一のログライブラリとして統一。

### 10. Connector.Config が []byte

**ファイル**: `storage/storage.go:386-391`

コネクタの設定が `[]byte` として格納され、型安全性が失われている。ストレージ層では設定内容を解釈できず、JSONの直列化/逆直列化をサーバー層で行う必要がある。

**代替案**: 型パラメータやインターフェースを使ってより型安全にするか、少なくとも `json.RawMessage` を使う。

---

## Part 8: 新規プロジェクトへの適用ガイド

### 推奨アーキテクチャ骨格

Dexの設計を基に、新規OAuth2/OIDCプロジェクト向けの改善版構成を提案する:

```
myproject/
├── cmd/
│   └── server/
│       └── main.go              # エントリーポイント
├── internal/                    # 外部非公開の全実装
│   ├── server/                  # HTTPサーバー・ハンドラー
│   │   ├── server.go
│   │   ├── oauth2.go
│   │   ├── discovery.go
│   │   ├── token.go
│   │   └── middleware.go
│   ├── provider/                # IDプロバイダ連携（Dexのconnector相当）
│   │   ├── provider.go          # インターフェース定義
│   │   ├── oidc/
│   │   ├── saml/
│   │   └── ldap/
│   ├── store/                   # 永続化レイヤー（Dexのstorage相当）
│   │   ├── store.go             # インターフェース群
│   │   ├── memory/
│   │   ├── postgres/
│   │   └── conformance/         # 適合テスト ← Dexから採用
│   ├── token/                   # トークン生成・検証
│   │   ├── signer.go            # Signerインターフェース
│   │   ├── jwt.go
│   │   └── rotation.go
│   └── config/                  # 設定パース・バリデーション
├── api/                         # 公開API定義（proto等）
├── pkg/                         # 外部公開ユーティリティ（最小限）
├── web/                         # フロントエンドアセット
├── go.mod
├── Makefile
└── .golangci.yaml
```

### 推奨パッケージ構成の改善点（Dexとの差異）

| Dexの設計 | 推奨改善 | 理由 |
|-----------|---------|------|
| トップレベルに `server/`, `connector/`, `storage/` | `internal/` 配下に配置 | APIの安定性保証。外部から依存されない |
| 巨大な `Storage` インターフェース | エンティティ別の小インターフェースを合成 | Interface Segregation |
| Connector登録のハードコード | `init()` + build tagsまたはDI | 結合度低減 |
| gorilla/mux | Go標準 `net/http` (1.22+) | 外部依存削減 |
| `pkg/errors` | 標準 `errors` のみ | 非推奨ライブラリ排除 |
| Generics未活用 | CRUD操作にgenerics | 型安全性・DRY |

### 現代的Go技法（Dexで未使用だが新規なら導入すべき）

1. **Genericsを活用したRepository パターン**:
```go
type Repository[T any, ID comparable] interface {
    Create(ctx context.Context, entity T) error
    Get(ctx context.Context, id ID) (T, error)
    Update(ctx context.Context, id ID, fn func(T) (T, error)) error
    Delete(ctx context.Context, id ID) error
    List(ctx context.Context) ([]T, error)
}
```

2. **`log/slog` の統一使用（Dexは移行中）**:
```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
    AddSource: true,
}))
```

3. **`errors.Join` によるマルチエラー（Go 1.20+）**

4. **`context.AfterFunc` の活用（Go 1.21+）**

5. **`testing.TB.Context()` の使用（Go 1.24+）**: Dexは既に使用中（`t.Context()`）

6. **構造化エラー型 + `errors.Is`/`errors.As` の一貫使用**:
```go
type NotFoundError struct {
    Resource string
    ID       string
}
func (e *NotFoundError) Error() string { ... }

// 使用側
var nfe *NotFoundError
if errors.As(err, &nfe) { ... }
```

7. **Go 1.22+ の `net/http` ルーティング**:
```go
mux := http.NewServeMux()
mux.HandleFunc("GET /api/clients/{id}", handleGetClient)
mux.HandleFunc("POST /token", handleToken)
```

### 最初に定義すべきインターフェース群

OAuth2/OIDCサーバーの新規開発で、以下のインターフェースを最初に定義することを推奨:

```go
// 1. トークン署名
type Signer interface {
    Sign(ctx context.Context, payload []byte) (string, error)
    VerificationKeys(ctx context.Context) ([]jose.JSONWebKey, error)
}

// 2. IDプロバイダ連携（Dexのconnector相当）
type PasswordAuthenticator interface {
    Authenticate(ctx context.Context, username, password string) (Identity, error)
}
type OAuthProvider interface {
    AuthorizationURL(state string, scopes []string) string
    Exchange(ctx context.Context, code string) (Identity, *oauth2.Token, error)
}
type RefreshableProvider interface {
    Refresh(ctx context.Context, identity Identity) (Identity, error)
}

// 3. ストレージ（エンティティ別）
type ClientStore interface {
    CreateClient(ctx context.Context, c Client) error
    GetClient(ctx context.Context, id string) (Client, error)
    // ...
}
type AuthorizationStore interface { /* ... */ }
type TokenStore interface { /* ... */ }
type SessionStore interface { /* ... */ }

// 4. ヘルスチェック
type HealthChecker interface {
    Check(ctx context.Context) error
}
```

---

## Part 9: 総合評価サマリー

### 5段階評価

| 観点 | 評価 | コメント |
|------|:----:|---------|
| **設計** | ★★★★☆ | Connector/Storage/Signerの3層抽象化は秀逸。Storageインターフェースの肥大化が減点 |
| **テスト** | ★★★★☆ | 適合テストパターンは秀逸。テストカバレッジ比率（19%）はやや低い。errcheck無効による見落とし懸念 |
| **ドキュメント** | ★★★☆☆ | READMEとコード内コメントは良質。GoDocは部分的。docs/は別リポジトリに移行済みで空に近い |
| **セキュリティ** | ★★★★☆ | PKCE、セキュリティヘッダー、X-Remote-*ストリッピング。Vault/OpenBao連携。SAMLコネクタがunmaintainedな点が懸念 |
| **拡張性** | ★★★★★ | コネクタ17種、ストレージ6種を同一インターフェースで管理。新規追加の手順が明確 |
| **Goらしさ** | ★★★★☆ | インターフェース設計、命名、パッケージ構成はidiomaticGoに近い。Generics未活用、一部レガシーパターン残存 |

### 最も学ぶべきトップ3の設計判断

1. **ストレージ適合テスト（Conformance Test）パターン** — インターフェースの契約をテストで形式化し、全実装の正しさを保証。新規プロジェクトに**そのまま適用**できる。

2. **Updater関数によるCAS更新** — ストレージ層のトランザクション詳細を隠蔽しつつ、呼び出し側に安全なread-modify-writeを提供。データ整合性の基盤。

3. **Connectorインターフェースの段階的分離** — 空のマーカーインターフェース + 小さな機能インターフェースの組み合わせ。サーバーは型アサーションで機能を判別。**Go Interface Segregation Principleの模範実装**。

### 最も注意すべきトップ3のアンチパターン

1. **39メソッドの巨大Storageインターフェース** — エンティティ追加のたびに全バックエンドを変更。新規プロジェクトでは小インターフェースの合成で対処。

2. **errcheck/staticcheckの無効化** — 品質ゲートの甘さ。未チェックのエラーが本番障害につながるリスク。新規プロジェクトでは最初から有効にする。

3. **JSONタグのバグ放置（Connector.Config `json:"email"`）** — 後方互換性維持のための妥協。APIスキーマは最初から正確に設計すべき教訓。

### フォークして使うべきか、参考にして自前実装すべきか

**結論: 参考にして自前実装すべき**

| 判断軸 | 評価 |
|--------|------|
| Dexをフォークする場合のメリット | 17コネクタ、6ストレージバックエンドが即座に利用可能 |
| Dexをフォークする場合のデメリット | アプリケーション構造がDexに束縛される。39メソッドStorageインターフェースの変更が困難。gorilla/mux, pkg/errors等のレガシー依存が付随 |
| 自前実装する場合のメリット | Generics活用、小インターフェース設計、modern Go技法を最初から採用可能。後方互換性の制約なし |
| 自前実装する場合のデメリット | コネクタ実装の工数。OAuth2/OIDCフロー実装の複雑さ |

**推奨アプローチ**:
1. Dexの**設計パターン**（適合テスト、Updater関数、Connector分離）を参考に自前の骨格を設計
2. Dexの**具体実装**（特にOAuth2フロー `server/oauth2.go`、鍵ローテーション `server/signer/rotation.go`）をリファレンスとして参照
3. 必要なコネクタは`coreos/go-oidc`や`golang.org/x/oauth2`を直接使って実装（Dexも内部でこれらを使用）
4. ストレージ層は最初からgenerics + 小インターフェースで設計し、適合テストを先に書く
