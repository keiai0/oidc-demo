# Ory Hydra 徹底分析レポート

> **対象リポジトリ:** https://github.com/ory/hydra
> **調査日:** 2026-02-27
> **調査対象コミット:** `55dadd5` (2026-02-21)
> **目的:** Go製OAuth2/OpenID Connect新規プロジェクトの設計参考資料

---

## Part 1: プロジェクト概要

> Ory HydraはOpenID Certified済みのOAuth 2.0/OpenID Connectサーバーであり、ユーザー管理を持たず外部のIDプロバイダーと連携する設計が最大の特徴。

### プロジェクトの目的・スコープ

- **種別:** サーバーアプリケーション（自己完結型のOAuth2/OIDCプロバイダー）
- **解決する課題:** OAuth2/OIDCの仕様準拠実装を提供し、ログイン/同意UIは外部アプリに委譲する
- **スコープ:**
  - OAuth 2.0 Authorization Framework (RFC 6749)
  - OAuth 2.0 Token Revocation (RFC 7009) / Introspection (RFC 7662)
  - PKCE (RFC 7636)
  - JWT Bearer Grant (RFC 7523)
  - Device Authorization Grant (RFC 8628)
  - Dynamic Client Registration (RFC 7591/7592)
  - Pushed Authorization Requests (RFC 9126)
  - OpenID Connect Core 1.0 / Discovery / Dynamic Registration
  - Front-Channel / Back-Channel Logout
- **スコープ外:** ユーザー管理・認証UI（Ory Kratosや自前実装に委譲）

### 設計哲学

README・コード・アーキテクチャから読み取れる設計原則:

1. **ユーザー管理の分離** — OAuth2サーバーは認証UIを持たず、Login/Consent Appと連携
2. **標準準拠** — OpenID Certified取得済み、400ページ超のRFC群を忠実に実装
3. **クラウドネイティブ** — 小バイナリ(5-15MB)、コンテナファースト、ステートレス設計
4. **最小依存** — OS依存なし（Java/Node/Ruby不要）
5. **マルチテナント対応** — NID（Network ID）によるデータ隔離
6. **低レイテンシ・高スループット** — パフォーマンスベンチマーク付き

### Goバージョン・主要依存ライブラリ

| カテゴリ | ライブラリ | 用途 |
|---------|-----------|------|
| **Go Version** | `go 1.26` | 最新 |
| **OAuth2コア** | `fosite` (内蔵) | OAuth2/OIDCプロトコル実装 |
| **CLI** | `spf13/cobra` | コマンドライン |
| **DB/ORM** | `ory/pop/v6` | DB抽象化・マイグレーション |
| **暗号** | `go-jose/go-jose/v3`, `golang-jwt/jwt/v5` | JWK/JWT |
| **HSM** | `ThalesGroup/crypto11`, `miekg/pkcs11` | ハードウェアセキュリティモジュール |
| **HTTP** | `rs/cors`, `urfave/negroni` | CORS・ミドルウェア |
| **テスト** | `stretchr/testify`, `go.uber.org/mock` | アサーション・モック |
| **可観測性** | `opentelemetry` (otel系), `sirupsen/logrus` | トレーシング・ロギング |
| **設定** | `knadh/koanf/v2` (ory/x経由) | 構造化設定管理 |
| **JSON操作** | `tidwall/gjson`, `tidwall/sjson` | 高速JSON読み書き |
| **キャッシュ** | `dgraph-io/ristretto/v2` | インメモリキャッシュ |
| **DB Driver** | `jackc/pgx/v5`, `go-sql-driver/mysql`, `mattn/go-sqlite3` | PostgreSQL/MySQL/SQLite |

### プロジェクトの成熟度

| 指標 | 値 |
|------|-----|
| GitHub Stars | 15,000+ (推定・OpenAI, Cisco, Klarna等が採用) |
| 最終コミット | 2026-02-21 (活発) |
| 直近コミッターの多様性 | 17名（直近100コミット） |
| OpenID Certification | 取得済み (v1.0.0) |
| ライセンス | Apache 2.0 |
| テストファイル数 | 190ファイル |
| ソースファイル数 | 579ファイル（生成コード除く） |
| go.mod直接依存 | 約60パッケージ |

---

## Part 2: アーキテクチャ分析

> Hydraは「Service Registry Pattern」を中心に据え、fosite（OAuth2コアライブラリ）をラップする多層アーキテクチャを採用。

### ディレクトリ構成

```
hydra/
├── main.go                  # エントリーポイント（cmd.Execute()のみ）
├── doc.go                   # パッケージドキュメント
├── cmd/                     # CLI定義（Cobra）
│   ├── root.go             #   ルートコマンド
│   ├── serve.go            #   サーバー起動（all/admin/public）
│   ├── server/             #   サーバーハンドラ・ミドルウェア
│   ├── cli/                #   CLIヘルパー
│   ├── cliclient/          #   CLIクライアント接続
│   └── clidoc/             #   CLIドキュメント自動生成
├── driver/                  # 依存性注入・レジストリ
│   ├── registry.go         #   Registry interface定義
│   ├── registry_sql.go     #   RegistrySQL実装（中心的DI容器）
│   ├── factory.go          #   RegistrySQL構築
│   ├── di.go               #   DI初期化ロジック
│   └── config/             #   アプリケーション設定
├── fosite/                  # OAuth2コアライブラリ（内蔵ライブラリ）
│   ├── oauth2.go           #   OAuth2Provider interface
│   ├── fosite.go           #   Fosite struct・ハンドラチェーン
│   ├── handler/            #   各フロー実装
│   │   ├── oauth2/         #     Authorization Code, Client Credentials等
│   │   ├── openid/         #     OpenID Connect拡張
│   │   ├── pkce/           #     PKCE
│   │   ├── par/            #     Pushed Authorization Request
│   │   ├── rfc7523/        #     JWT Bearer Grant
│   │   ├── rfc8628/        #     Device Authorization
│   │   └── verifiable/     #     Verifiable Credentials
│   ├── token/              #   トークン生成戦略
│   │   ├── hmac/           #     HMAC-SHA512/256
│   │   └── jwt/            #     JWT
│   ├── compose/            #   Factory/Composition パターン
│   ├── storage/            #   ストレージ抽象
│   └── internal/           #   MockGen生成モック
├── fositex/                 # fosite拡張・設定統合
├── oauth2/                  # OAuth2エンドポイントハンドラ
│   ├── handler.go          #   HTTPルーティング
│   ├── session.go          #   セッションモデル
│   └── trust/              #   JWT Grant Trust
├── client/                  # OAuth2クライアント管理
├── consent/                 # Login/Consentフロー管理
│   └── strategy_default.go #   フロー戦略実装
├── flow/                    # フロー状態機械
│   ├── flow.go             #   Flow struct・状態定義
│   └── state_transition.go #   状態遷移ルール
├── jwk/                     # JSON Web Key管理
├── aead/                    # AEAD暗号（AES-GCM / XChaCha20）
├── hsm/                     # Hardware Security Module統合
├── health/                  # ヘルスチェック
├── persistence/             # 永続化層
│   └── sql/                #   SQL実装（PostgreSQL/MySQL/CockroachDB/SQLite）
├── internal/                # 内部パッケージ
│   ├── testhelpers/        #   テストユーティリティ
│   ├── mock/               #   モック
│   └── httpclient/         #   生成SDKクライアント
├── x/                       # 共有ユーティリティ
│   ├── fosite_storer.go    #   統合ストレージinterface
│   ├── events/             #   イベント定義
│   └── oauth2cors/         #   CORS設定
├── spec/                    # OpenAPI仕様
├── test/                    # E2E・適合性テスト
│   ├── e2e/                #   E2Eテストスクリプト
│   └── conformance/        #   OpenID Connect適合性テスト
├── cypress/                 # Cypress E2Eテスト
├── scripts/                 # ビルド・運用スクリプト
└── .github/workflows/       # CI/CD
```

### 各パッケージの責務と依存方向

```
                          ┌─────────────────┐
                          │   cmd/server    │  HTTPサーバー起動
                          └────────┬────────┘
                                   │
                          ┌────────▼────────┐
                          │     driver      │  依存性注入（Registry）
                          │  registry_sql   │  全コンポーネントの統合点
                          └───┬──┬──┬──┬────┘
                              │  │  │  │
              ┌───────────────┘  │  │  └───────────────┐
              │                  │  │                   │
     ┌────────▼───────┐  ┌──────▼──▼─────┐   ┌────────▼───────┐
     │    consent     │  │    oauth2     │   │     client     │
     │ Login/Consent  │  │  Endpoints   │   │   CRUD/Auth    │
     │   Strategy     │  │  Token/Auth  │   │   Validation   │
     └────────┬───────┘  └──────┬───────┘   └────────────────┘
              │                 │
     ┌────────▼───────┐  ┌─────▼──────────┐
     │     flow       │  │   fositex      │  fosite設定統合
     │  State Machine │  └─────┬──────────┘
     └────────────────┘        │
                          ┌────▼──────────┐
                          │    fosite     │  OAuth2/OIDCコアライブラリ
                          │  (embedded)   │  プロトコル実装
                          └───────────────┘
                                │
     ┌──────────┬───────────────┼───────────────┐
     │          │               │               │
┌────▼───┐ ┌───▼────┐  ┌───────▼──────┐  ┌─────▼────┐
│  jwk   │ │  aead  │  │ persistence  │  │   hsm    │
│ JWK管理│ │ 暗号化  │  │   SQL永続化   │  │ HSM統合  │
└────────┘ └────────┘  └──────────────┘  └──────────┘
```

**依存方向の原則:** 上位→下位の一方向依存。`driver`パッケージがすべてを統合し、各ドメインパッケージは互いに直接依存しない（Registryを通じて間接的にアクセス）。

### 設計パターン

#### 1. Service Registry Pattern（中心的パターン）

**場所:** `driver/registry.go`, `driver/registry_sql.go`

Hydra最大の設計特徴。`RegistrySQL`が全サービスのファクトリ兼サービスロケータとして機能する。

```go
// driver/registry.go — Registryインターフェース（簡略版）
type Registry interface {
    client.Registry
    consent.Registry
    jwk.Registry
    oauth2.Registry
    persistence.Provider
    // ... 他のプロバイダー
}
```

```go
// driver/registry_sql.go — 遅延初期化パターン
func (m *RegistrySQL) KeyManager() jwk.Manager {
    if m.keyManager == nil {
        softwareKeyManager := &sql.JWKPersister{D: m}
        if m.Config().HSMEnabled() {
            hardwareKeyManager := hsm.NewKeyManager(m.HSMContext(), m.Config())
            m.keyManager = jwk.NewManagerStrategy(hardwareKeyManager, softwareKeyManager)
        } else {
            m.keyManager = softwareKeyManager
        }
    }
    return m.keyManager
}
```

**評価:** 遅延初期化により循環依存を回避し、条件分岐（HSM有無等）も自然に扱える。ただしサービスロケータはアンチパターンとも言われ、依存関係が暗黙的になる。

#### 2. Chain of Responsibility（ハンドラチェーン）

**場所:** `fosite/fosite.go`, `fosite/handler.go`

各OAuth2エンドポイントに対して、複数のハンドラが順次処理する。

```go
// fosite/fosite.go
type AuthorizeEndpointHandlers []AuthorizeEndpointHandler
type TokenEndpointHandlers    []TokenEndpointHandler

// 重複防止付きAppend
func (a *AuthorizeEndpointHandlers) Append(h AuthorizeEndpointHandler) {
    for _, this := range *a {
        if reflect.TypeOf(this) == reflect.TypeOf(h) {
            return // 同一型のハンドラは無視
        }
    }
    *a = append(*a, h)
}
```

**評価:** OAuth2の複数フロー（Authorization Code, Implicit, Client Credentials等）を統一的に処理するのに適した設計。`reflect.TypeOf`による重複防止はエレガント。

#### 3. Strategy Pattern

**場所:** `fosite/handler/oauth2/strategy.go`, `consent/strategy.go`

トークン生成やLogin/Consentフローの戦略を差し替え可能にする。

```go
// fosite/handler/oauth2/strategy.go — トークン生成戦略
type CoreStrategy interface {
    AuthorizeCodeStrategy
    AccessTokenStrategy
    RefreshTokenStrategy
}

// consent/strategy.go — フロー戦略
type Strategy interface {
    HandleOAuth2AuthorizationRequest(ctx, w, r, req) (*flow.Flow, error)
    HandleOpenIDConnectLogout(ctx, w, r) (*flow.LogoutResult, error)
}
```

#### 4. Factory / Composition Pattern

**場所:** `fosite/compose/compose.go`

ファクトリ関数群を合成して完全なOAuth2Providerを構築する。

```go
// fosite/compose/compose.go
type Factory func(config fosite.Configurator, storage fosite.Storage, strategy interface{}) interface{}

func Compose(config *fosite.Config, storage fosite.Storage, strategy interface{},
    factories ...Factory) fosite.OAuth2Provider {
    f := fosite.NewOAuth2Provider(storage, config)
    for _, factory := range factories {
        res := factory(config, storage, strategy)
        if ah, ok := res.(fosite.AuthorizeEndpointHandler); ok {
            config.AuthorizeEndpointHandlers.Append(ah)
        }
        if th, ok := res.(fosite.TokenEndpointHandler); ok {
            config.TokenEndpointHandlers.Append(th)
        }
        // ... 他のハンドラタイプも同様
    }
    return f
}
```

#### 5. Provider Interface Pattern（設定の依存性注入）

**場所:** `fosite/config.go`

設定値ごとに小さなProviderインターフェースを定義し、必要な設定のみを依存として要求する。

```go
// fosite/config.go
type AccessTokenLifespanProvider interface {
    GetAccessTokenLifespan(ctx context.Context) time.Duration
}

type EnforcePKCEProvider interface {
    GetEnforcePKCE(ctx context.Context) bool
}

// 全設定を集約
type Configurator interface {
    AccessTokenLifespanProvider
    EnforcePKCEProvider
    // ... 数十個のProvider
}
```

**評価:** Interface Segregation Principleの極端な適用。各ハンドラは必要な設定Providerだけを要求でき、テスト時のモック範囲が最小化される。`context.Context`を引数に取ることでリクエスト単位の設定変更も可能。

#### 6. State Machine Pattern

**場所:** `flow/flow.go`, `flow/state_transition.go`

Login→Consent→Token発行のフローを状態機械で管理する。

```
LOGIN_UNUSED → LOGIN_USED → CONSENT_UNUSED → CONSENT_USED → [完了]
                         ↘ LOGIN_ERROR → [中断]
                                        ↘ CONSENT_ERROR → [中断]
```

#### インターフェース設計の方針

- **小さなインターフェース:** 1-3メソッドのProviderインターフェースが多数（Go Code Review Commentsの推奨に沿う）
- **埋め込みによる合成:** `Registry`は`client.Registry`, `consent.Registry`等を埋め込んで構成
- **具象型への依存最小化:** ハンドラは具体的なストレージ型ではなくインターフェースに依存

#### 構造体の設計

- **コンストラクタ:** `New〜()`関数による初期化（例: `fosite.NewOAuth2Provider()`）
- **Functional Options:** `driver/factory.go`で`configx.OptionModifier`を使用
- **ゼロ値安全性:** `RegistrySQL`は遅延初期化のため、ゼロ値でも安全

#### エラーハンドリングのパターン

```go
// fosite/errors.go — RFC 6749準拠エラー型
type RFC6749Error struct {
    ErrorField       string   // OAuth2エラーコード（invalid_request等）
    DescriptionField string   // ユーザー向け説明
    HintField        string   // 開発者向けヒント
    CodeField        int      // HTTPステータスコード
    DebugField       string   // デバッグ情報（本番では非公開）
}

// Fluent API
err := fosite.ErrInvalidRequest.
    WithHint("The redirect_uri is malformed").
    WithDebug("expected https scheme").
    WithWrap(originalError)
```

- `github.com/pkg/errors`によるスタックトレース保持
- `RFC6749Error`はJSON直列化時にDebug情報の公開をコントロール
- i18n対応（`WithLocalizer`）

#### context.Context の活用

- 全ストレージ操作の第1引数に`ctx`
- ネットワークID（マルチテナント）の伝搬
- 設定のリクエストスコープ解決（`GetAccessTokenLifespan(ctx)`）
- OpenTelemetryトレーシングのスパン伝搬
- トランザクション管理（`popx.Transaction`内でctx経由）

---

## Part 3: Go実装の詳細分析

> 全体的にGo標準のベストプラクティスに高い水準で準拠。ただしgenerics未使用、logrusレガシー等の改善余地あり。

### パッケージ構成とGoプロジェクトレイアウト

- **`cmd/`**: CLI定義 — Go標準レイアウト準拠
- **`internal/`**: 外部非公開パッケージ — `testhelpers/`, `mock/`, `httpclient/`（生成SDK）
- **`driver/`**: 独自だがDI層として明確な役割
- **`fosite/`**: 内蔵ライブラリ（元は別リポジトリ`ory/fosite`だったものをmonorepo化）
- **`x/`**: 共有ユーティリティ（Ory共通パターン）
- **`persistence/sql/`**: 永続化実装

**事実:** Go公式の`/pkg`ディレクトリは使用していない。ドメイン名によるフラットなパッケージ分割（`client/`, `consent/`, `oauth2/`, `jwk/`等）。

### internal パッケージの活用

- `internal/testhelpers/` — テスト用ドライバ・サーバー構築ヘルパー
- `internal/mock/` — テスト用モック
- `internal/httpclient/` — OpenAPIから生成されたGoクライアントSDK
- `internal/config/` — 設定スキーマ
- `internal/kratos/` — Kratos連携インターフェース
- `internal/certification/` — OpenID Connect適合性テストデータ

**評価:** `internal`の活用は適切。生成コード（httpclient）やテストヘルパーを外部非公開にしている。

### エクスポート/非エクスポートの判断基準

- **エクスポート:** インターフェース、構造体（DB永続化されるモデル）、ハンドラ
- **非エクスポート:** ヘルパー関数、内部状態、設定キー定数（`config`パッケージ内でconst）
- **特徴:** `fosite`パッケージはライブラリとして設計されているため、ほぼ全てエクスポート

### 命名規則

| 対象 | 規則 | 例 |
|------|------|-----|
| パッケージ名 | 小文字単語 | `client`, `consent`, `flow`, `aead`, `fositex` |
| インターフェース名 | 名詞/er接尾辞 | `Manager`, `Strategy`, `Requester`, `Configurator` |
| Provider IF名 | `〜Provider` | `AccessTokenLifespanProvider`, `EnforcePKCEProvider` |
| レシーバ名 | 1文字 | `(m *RegistrySQL)`, `(f *Fosite)`, `(c *Client)` |
| テストヘルパー | `Must〜` / `New〜` | `MustEnsureRegistryKeys()`, `NewConfigurationWithDefaults()` |
| ファクトリ関数 | `New〜` | `NewOAuth2Provider()`, `NewKeyManager()` |
| 定数 | `Key〜` / `Err〜` | `KeyBCryptCost`, `ErrInvalidRequest` |

**評価:** Go Code Review Commentsの推奨に概ね準拠。レシーバ名は1文字で統一されている。

### ファイル分割の粒度

- **1ファイル1責務:** `handler.go`（HTTPルーティング）、`session.go`（セッション）、`registry.go`（DI定義）
- **テストは同一パッケージ:** `handler_test.go`は同ディレクトリに配置
- **ストレージは種類別:** `persister_client.go`, `persister_oauth2.go`, `persister_consent.go`等
- **スナップショットテスト:** `.snapshots/`ディレクトリに期待値JSON

### GoDocコメント

- **インターフェース:** 一部にGoDocあり、但し充実度は中程度
- **エクスポート関数:** 主要なものにはコメントあり
- **コード内コメント:** 複雑なビジネスロジック（consent/strategy_default.go）に理由コメントあり
- **改善余地:** 一部のパブリックAPIにはGoDocが不足

### 並行処理パターン

- **goroutine:** 直接的なgoroutine使用は少ない（HTTPサーバーが暗黙的に処理）
- **sync:** キャッシュやレジストリの初期化でsync関連を使用
- **context.Context:** 全操作でキャンセレーション・タイムアウト対応
- **errgroup/sourcegraph/conc:** 依存関係に`sourcegraph/conc`あり（並行処理ユーティリティ）
- **特徴:** OAuth2サーバーの性質上、リクエスト単位の処理が主であり、複雑な並行処理は少ない

### go generate の活用

```go
// fosite/generate.go — モック生成
//go:generate mockgen -package internal -destination internal/authorize_endpoint_handler.go ...

// go.mod tool ディレクティブ
tool (
    github.com/go-swagger/go-swagger/cmd/swagger  // OpenAPI生成
    github.com/mikefarah/yq/v4                     // YAML処理
    go.uber.org/mock/mockgen                        // モック生成
    golang.org/x/tools/cmd/goimports               // import整理
)
```

**活用状況:** モック生成（mockgen）、SDK生成（go-swagger）、YAML処理（yq）に使用。Go 1.24+の`tool`ディレクティブを採用。

---

## Part 4: テスト戦略

> 多層的テスト戦略（Unit → Integration → E2E → 適合性テスト）を採用。testify + gomock + dockertest + Cypressの組み合わせ。

### テストフレームワーク・ライブラリ

| ライブラリ | 用途 |
|-----------|------|
| `stretchr/testify` | アサーション（`assert`, `require`） |
| `go.uber.org/mock/gomock` | インターフェースモック生成 |
| `ory/dockertest/v3` | テスト用Dockerコンテナ管理 |
| `net/http/httptest` | HTTPサーバーモック |
| `bradleyjkemp/cupaloy/v2` | スナップショットテスト |
| `tidwall/gjson` | JSONアサーション |
| `Cypress` | ブラウザE2Eテスト |

### テストの種類と構成

| 種類 | 実行方法 | 対象 |
|------|---------|------|
| **Unit** | `go test -short -tags sqlite ./...` | SQLiteインメモリ、外部依存なし |
| **Integration** | `make test`（Docker必要） | PostgreSQL 18, MySQL 9.6, CockroachDB |
| **E2E** | `make e2e`（Cypress） | ブラウザベースのOAuth2フロー |
| **適合性** | `./test/conformance/test.sh` | OpenID Connect Certified準拠確認 |
| **HSM** | `go test -tags=hsm ./...` | SoftHSM2 PKCS#11テスト |
| **ベンチマーク** | `go test -bench=. ./oauth2/...` | パフォーマンス計測・プロファイリング |

### テーブル駆動テストの採用状況

広範に使用。典型例:

```go
// fosite/handler/oauth2/flow_authorize_code_auth_test.go
for _, c := range []struct {
    handler     AuthorizeExplicitGrantHandler
    areq        *fosite.AuthorizeRequest
    description string
    expectErr   error
    expect      func(t *testing.T, areq *fosite.AuthorizeRequest, aresp *fosite.AuthorizeResponse)
}{
    {description: "should fail because not responsible", ...},
    {description: "should fail because redirect_uri mismatch", ...},
    // ...
} {
    t.Run("case="+c.description, func(t *testing.T) { ... })
}
```

### モック・スタブの作り方

1. **mockgen自動生成:** `fosite/internal/`に全インターフェースのモック
2. **手動モック:** `internal/mock/`にHydra固有モック
3. **テスト用レジストリ:** `internal/testhelpers/driver.go`の`NewRegistryMemory()`
4. **RegistryModifier:** テスト時にレジストリの挙動を差し替え

```go
// Makefile — モック生成コマンド
mocks:
    go tool mockgen -package oauth2_test \
        -destination oauth2/oauth2_provider_mock_test.go \
        github.com/ory/fosite OAuth2Provider
```

### テストヘルパー・フィクスチャ管理

**テストヘルパー** (`internal/testhelpers/`):
- `NewConfigurationWithDefaults()` — BCryptコスト4、trace log等のテスト用デフォルト
- `NewRegistryMemory()` — SQLiteインメモリDB
- `ConnectDatabases()` — 並列マルチDB接続（dockertest）
- `MustEnsureRegistryKeys()` — JWK初期化
- `NewIDToken()` — テスト用IDトークン生成

**フィクスチャ:**
- `flow/fixtures/` — フロー状態のJSONフィクスチャ
- `oauth2/fixtures/` — OAuth2レスポンスフィクスチャ
- `.snapshots/` — スナップショットテスト期待値

### テストカバレッジ

- Codecov連携済み（`internal/httpclient`除外）
- `goveralls`コマンドも依存に含む
- CI上で`-coverprofile coverage.out`を出力

---

## Part 5: CI/CD・開発ツール

> GitHub Actionsで6種類のジョブを並列実行。Goreleaserによるマルチプラットフォームリリース。

### CI構成（GitHub Actions）

**`.github/workflows/ci.yaml` — メインパイプライン:**

| ジョブ | 内容 |
|--------|------|
| `oidc-conformity` | OpenID Connect適合性テスト |
| `test` | Unit/Integration テスト + Lint + Security scan |
| `test-hsm` | SoftHSM2テスト |
| `test-e2e` | Cypress E2E（4 DB × 2モード = 8マトリクス） |
| `docs-cli` | CLIドキュメント自動生成 |
| `release` | Goreleaserリリース（タグ時のみ） |

**その他ワークフロー:**
- `format.yml` — コードフォーマット検証
- `cve-scan.yaml` — Grype/Trivy/Kubescape/Dockle/Hadolintによるセキュリティスキャン
- `codeql-analysis.yml` — CodeQL静的解析（Go, JavaScript）
- `conventional_commits.yml` — コミットメッセージ規約チェック
- `stale.yml` — 非活動Issue自動クローズ

### Linter設定

```yaml
# .golangci.yml
version: "2"
linters:
  enable:
    - errcheck        # エラー未処理チェック
    - ineffassign     # 無効な代入チェック
    - staticcheck     # 静的解析
    - unused          # 未使用コードチェック
  settings:
    staticcheck:
      checks:
        - "-SA1019"   # deprecated警告は除外
  exclusions:
    rules:
      - path: '_test\.go'
        linters:
          - gosec     # テストコードではgosec除外
      - path: "internal/httpclient"
        linters:
          - errcheck  # 生成コードではerrcheck除外
```

**評価:** 最小限のLinter構成。`govet`, `gocritic`, `gocyclo`等は有効化されていない。新規プロジェクトではより厳格な設定が推奨される。

### Makefile タスク

| タスク | 内容 |
|--------|------|
| `quicktest` | `-short -tags sqlite`での高速テスト |
| `quicktest-hsm` | Docker内HSMテスト |
| `test` | DB起動 + 全テスト |
| `test-resetdb` | テスト用DBコンテナ再起動 |
| `e2e` | 全E2Eテスト |
| `format` | goimports + prettier + copyright headers |
| `lint` | golangci-lint v2.10.1 |
| `mocks` | mockgen実行 |
| `sdk` | OpenAPIからGoクライアント生成 |
| `install` | `go install` |

### リリース・バージョニング

- **Goreleaser:** マルチプラットフォームビルド（Linux/macOS/Windows × AMD64/ARM64/i386）
- **Docker:** Alpine + Distroless Staticの2イメージ
- **バージョニング:** Git tag (`v2.x.x`) によるセマンティックバージョニング
- **ビルド情報:** `config.Commit`, `config.Version`, `config.Date`をldflags注入

### コミットメッセージ規約

Conventional Commits準拠（CIで検証）:
- `feat:` / `fix:` / `chore:` / `docs:` / `refactor:`
- `autogen(sdk):` — 自動生成

### PR/Issueテンプレート

- `BUG-REPORT.yml` — 再現手順・ログ付きバグ報告
- `FEATURE-REQUEST.yml` — 機能提案テンプレート
- `DESIGN-DOC.yml` — 設計ドキュメントテンプレート

---

## Part 6: 参考にすべき点（✅ Adopt）

> 具体的なコード例・ファイルパスと共に、新規プロジェクトで採用すべき設計判断を解説。

### 1. Provider Interfaceによる設定の依存性注入

**場所:** `fosite/config.go`

```go
type AccessTokenLifespanProvider interface {
    GetAccessTokenLifespan(ctx context.Context) time.Duration
}
```

**なぜ優れているか:**
- 各コンポーネントが必要な設定のみに依存し、テスト時のモックが最小化される
- `ctx`引数によりリクエスト単位で設定を変更可能（マルチテナント等）
- Interface Segregation Principleの実践的な適用例
- **Go Code Review Commentsの「Accept interfaces, return structs」に合致**

**適用方法:** 設定構造体をそのまま渡すのではなく、`〜Provider`インターフェースを定義して依存を宣言する。

### 2. Compose/Factory Patternによるモジュラー構成

**場所:** `fosite/compose/compose.go`, `fositex/config.go`

```go
func Compose(config *fosite.Config, storage fosite.Storage, strategy interface{},
    factories ...Factory) fosite.OAuth2Provider
```

**なぜ優れているか:**
- 必要なOAuth2フロー（Authorization Code, Client Credentials等）を選択的に有効化
- 新しいフローの追加が既存コードの変更なしに可能（Open-Closed Principle）
- ランタイム型アサーションで1つのファクトリが複数のハンドラ型を返せる

**適用方法:** OAuth2フローごとにFactory関数を定義し、`Compose`で合成する。不要なフロー（Implicitフロー等）を除外しやすい。

### 3. RFC 6749準拠エラー型

**場所:** `fosite/errors.go`

```go
type RFC6749Error struct {
    ErrorField       string
    DescriptionField string
    HintField        string
    CodeField        int
    DebugField       string
}

// Fluent API
err := fosite.ErrInvalidRequest.
    WithHint("The redirect_uri is missing").
    WithDebug("expected parameter 'redirect_uri' in request body")
```

**なぜ優れているか:**
- OAuth2仕様のエラーコード体系に完全準拠
- Fluent APIで開発者フレンドリー
- `DebugField`は本番では非公開にでき、セキュリティ上安全
- `WithWrap`で元エラーのスタックトレースも保持
- i18n対応可能

**適用方法:** そのまま自プロジェクトに取り入れるべきパターン。OAuth2エラーは仕様で定められているため、このエラー型設計は参考になる。

### 4. 状態機械によるフロー管理

**場所:** `flow/flow.go`, `flow/state_transition.go`

**なぜ優れているか:**
- Login→Consent→Token発行の複雑なフローを明示的な状態遷移で管理
- 不正な状態遷移を型安全に防止
- デバッグ時にフローの現在状態が明確

**適用方法:** OAuth2のAuthorization Code Flowは複数ステップの対話が必要。状態機械パターンで管理することで、不正なフロー操作（CSRFや状態改ざん等）を構造的に防止できる。

### 5. AEAD暗号化による保存時暗号化 + 鍵ローテーション

**場所:** `aead/aesgcm.go`, `aead/xchacha20.go`

```go
type Cipher interface {
    Encrypt(ctx context.Context, plaintext []byte, additionalData []byte) (string, error)
    Decrypt(ctx context.Context, ciphertext string, additionalData []byte) ([]byte, error)
}
```

**なぜ優れているか:**
- JWKやフロー状態等の機密データを保存時に暗号化
- AES-GCMとXChaCha20Poly1305の2実装を提供
- 鍵ローテーション対応（暗号化は新鍵、復号は全鍵で試行）
- Additional Authenticated Data (AAD)による認証付き暗号

**適用方法:** 新規プロジェクトでも秘密鍵やトークンのDB保存時にはAEAD暗号化を標準装備すべき。

### 6. NID（Network ID）によるマルチテナントデータ隔離

**場所:** `persistence/sql/persister.go`

```go
func (p *BasePersister) QueryWithNetwork(ctx context.Context) *pop.Query {
    return p.Connection(ctx).Where("nid = ?", p.NetworkID(ctx))
}
```

**なぜ優れているか:**
- 全テーブルにNIDカラムを持ち、クエリレベルで完全隔離
- `context.Context`経由でNIDを伝搬するため、ビジネスロジック側で意識不要
- 将来のマルチテナント化に対応

### 7. テストヘルパーの設計

**場所:** `internal/testhelpers/driver.go`

```go
func NewConfigurationWithDefaults() []configx.OptionModifier {
    return []configx.OptionModifier{
        configx.SkipValidation(),
        configx.WithValues(map[string]any{
            config.KeyBCryptCost: 4,     // テスト高速化
            config.KeyLogLevel:   "trace",
        }),
    }
}
```

**なぜ優れているか:**
- テスト用のデフォルト設定（BCryptコスト4等）で高速化
- `NewRegistryMemory()`で外部依存なしのテストが可能
- `ConnectDatabases()`で複数DBに対する並列テスト

### 8. セキュリティ面の実装

- **定数時間比較:** `fosite/token/hmac/hmacsha.go`でトークン比較にconstant-time
- **最小エントロピー検証:** トークン生成時にシークレットの最小長を検証
- **bcryptハッシュ:** クライアントシークレットはbcryptで保存
- **Debug情報の非公開:** RFC6749Errorで本番時にDebugFieldを隠蔽
- **HSM対応:** PKCS#11でハードウェアに鍵を格納可能
- **CVEスキャン:** CI/CDで5種類のセキュリティスキャナーを実行

### 9. OpenAPI駆動の開発

**場所:** `spec/`, `internal/httpclient/`

- OpenAPI仕様からGoクライアントSDKを自動生成
- API仕様とコードの乖離を防止
- テストでも生成SDKを使用してE2E検証

---

## Part 7: 参考にすべきでない点（⚠️ Avoid）

> 批判ではなく「コンテキストが異なる」「より良い選択肢がある」という視点。

### 1. Service Locator的なRegistryパターン

**場所:** `driver/registry_sql.go`

**問題:**
- `RegistrySQL`が全サービスへのアクセサーを提供するため、依存関係が暗黙的
- 600行超の巨大ファイルに成長し、新機能追加のたびに肥大化
- テスト時に不必要なサービスまで初期化される可能性

**代替案:**
- **Wire（google/wire）** や **Fx（uber-go/fx）** によるコンパイルタイムDI
- 明示的なコンストラクタインジェクション
- 新規プロジェクトでは`wire`で依存グラフを静的に検証するほうが安全

### 2. logrus の使用

**場所:** `go.mod` — `github.com/sirupsen/logrus v1.9.3`

**問題:**
- logrusはメンテナンスモード（作者が新機能開発を終了宣言済み）
- 構造化ロギングの標準が`log/slog`（Go 1.21+）に移行

**代替案:**
- `log/slog`（標準ライブラリ、Go 1.21+）
- `slog`はゼロ割当、構造化ロギング、ハンドラパターンを標準提供
- 新規プロジェクトでは`slog`一択

### 3. github.com/pkg/errors の使用

**場所:** 多数のファイルで`errors.WithStack()`, `errors.Wrap()`

**問題:**
- `pkg/errors`はアーカイブ済み（メンテナンス終了）
- Go 1.13+の`errors.Is()`, `errors.As()`, `fmt.Errorf("%w", err)`で多くのケースをカバー

**代替案:**
- 標準`errors`パッケージ + `fmt.Errorf("context: %w", err)`
- スタックトレースが必要なら`slog`のエラーログで代替

### 4. reflect.TypeOf による重複防止

**場所:** `fosite/fosite.go`

```go
func (a *AuthorizeEndpointHandlers) Append(h AuthorizeEndpointHandler) {
    for _, this := range *a {
        if reflect.TypeOf(this) == reflect.TypeOf(h) {
            return
        }
    }
    *a = append(*a, h)
}
```

**問題:**
- ランタイムリフレクションに依存
- 型の一致判定がコンパイル時に検証されない
- genericsが使えるGoバージョンでは不要

**代替案:**
- ハンドラにID/名前メソッドを持たせて文字列比較
- Go genericsを活用した型安全なコレクション

### 5. Compose関数での `interface{}` 多用

**場所:** `fosite/compose/compose.go`

```go
type Factory func(config fosite.Configurator, storage fosite.Storage,
    strategy interface{}) interface{}
```

**問題:**
- `strategy`と戻り値が`interface{}`で型安全性がない
- ランタイム型アサーションに依存し、コンパイル時エラー検出不可
- 設定ミスがパニックではなく静かな無視になる可能性

**代替案:**
- Go genericsによる型パラメータ化
- インターフェース制約型の活用
- 新規プロジェクトではgenericsでFactory patternを型安全にすべき

### 6. Pop ORM の使用

**場所:** `go.mod` — `github.com/ory/pop/v6`

**問題:**
- Ory独自フォーク版のPop ORM
- コミュニティが小さく、ドキュメントが限定的
- マイグレーション管理がPop独自フォーマット

**代替案:**
- **sqlc** — SQLファーストのコード生成（型安全、高パフォーマンス）
- **Ent** — Facebook製のGoスキーマファーストORM
- **sqlx** — 軽量SQL拡張
- **goose** / **atlas** — マイグレーション管理
- 新規プロジェクトでは`sqlc` + `goose`の組み合わせが推奨

### 7. Linter設定が保守的すぎる

**場所:** `.golangci.yml`

**問題:**
- 4つのLinterのみ有効（errcheck, ineffassign, staticcheck, unused）
- `govet`, `gocritic`, `gocyclo`, `gosec`, `exhaustive`等が無効
- `SA1019`（deprecated）警告を全面除外

**代替案:**
```yaml
# 新規プロジェクト推奨設定
linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - unused
    - gosec
    - gocritic
    - exhaustive   # enumの網羅性チェック
    - nilerr       # nilエラーチェック
    - errorlint    # errors.Is/As正しい使用
    - bodyclose    # HTTPレスポンスBody閉じ忘れ
```

### 8. go-jose v3 の使用

**場所:** `go.mod` — `github.com/go-jose/go-jose/v3`

**問題:**
- v4が既にリリース済み
- v3は古いAPI設計

**代替案:**
- `github.com/go-jose/go-jose/v4` にアップグレード
- v4はセキュリティ面の改善あり

### 9. 複数のJWTライブラリ併用

**場所:** `go.mod`

```
github.com/cristalhq/jwt/v4
github.com/golang-jwt/jwt/v5
fosite/token/jwt (独自実装)
```

**問題:**
- 3つのJWTライブラリが混在しており、メンテナンスコストと混乱のリスク

**代替案:**
- `golang-jwt/jwt/v5`に統一
- または`go-jose/go-jose/v4`のJWT機能に統一

### 10. fosite monorepo化に伴う複雑性

**場所:** `go.mod`のreplace directive

```go
replace (
    github.com/ory/hydra-client-go/v2 => ./internal/httpclient
    github.com/ory/x => ./oryx
)
```

**問題:**
- 元々別リポジトリだった`fosite`と`ory/x`をmonorepo内にコピー
- `replace`ディレクティブによるローカルパス置換が必要
- 外部から見た依存関係が不明確

**新規プロジェクトでの教訓:**
- 最初からmonorepoまたはマルチモジュールで設計する
- 後からのmonorepo化は大きな技術的負債

---

## Part 8: 新規プロジェクトへの適用ガイド

> Hydraの設計から抽出した骨格と、現代的な改善を加えた推奨構成。

### 取り入れるべきアーキテクチャ骨格

1. **ドメイン分割:** `client/`, `consent/`, `oauth2/`, `jwk/`, `flow/`のように、OAuth2の概念ごとにパッケージ分割
2. **Registryパターン（改善版）:** wireによるコンパイルタイムDIに置き換え
3. **fosite Compose Pattern:** ファクトリ合成によるフロー選択的有効化
4. **Provider Interface:** 設定の依存性注入（そのまま採用）
5. **状態機械:** フロー管理の状態遷移パターン（そのまま採用）
6. **AEAD暗号化:** 保存時暗号化 + 鍵ローテーション（そのまま採用）

### 推奨パッケージ構成

```
myproject/
├── cmd/
│   └── server/
│       └── main.go              # エントリーポイント
├── internal/
│   ├── oauth2/                  # OAuth2プロトコル実装
│   │   ├── handler.go          #   エンドポイントハンドラ
│   │   ├── provider.go         #   OAuth2Provider実装
│   │   ├── flow/               #   各フローのハンドラ
│   │   │   ├── authcode.go
│   │   │   ├── clientcred.go
│   │   │   ├── refresh.go
│   │   │   └── device.go
│   │   ├── token/              #   トークン生成戦略
│   │   │   ├── strategy.go     #     Strategy interface
│   │   │   ├── jwt.go
│   │   │   └── opaque.go
│   │   └── session.go          #   セッション管理
│   ├── oidc/                    # OpenID Connect拡張
│   │   ├── claims.go
│   │   ├── discovery.go
│   │   └── userinfo.go
│   ├── client/                  # クライアント管理
│   │   ├── model.go
│   │   ├── repository.go       #   Repositoryインターフェース
│   │   ├── service.go          #   ビジネスロジック
│   │   └── validator.go
│   ├── consent/                 # Login/Consentフロー
│   │   ├── flow.go             #   状態機械
│   │   ├── strategy.go         #   Strategy interface
│   │   └── handler.go
│   ├── jwk/                     # JWK管理
│   │   ├── manager.go
│   │   └── rotator.go
│   ├── crypto/                  # 暗号ユーティリティ
│   │   ├── aead.go
│   │   └── hash.go
│   ├── storage/                 # 永続化層
│   │   ├── postgres/           #   PostgreSQL実装
│   │   └── migrations/         #   マイグレーション
│   ├── config/                  # 設定管理
│   │   └── provider.go         #   Provider interfaces
│   └── server/                  # HTTPサーバー・ミドルウェア
│       ├── server.go
│       └── middleware.go
├── pkg/                         # 公開API（SDKとして利用可能な型）
│   ├── errors/                  # RFC 6749エラー型
│   └── model/                   # 公開データモデル
├── api/
│   └── openapi.yaml             # OpenAPI仕様
├── go.mod
├── go.sum
├── Makefile
├── .golangci.yml
└── Dockerfile
```

### 使っていないが新規なら導入すべき現代的Go技法

| 技法 | Hydraの状態 | 推奨 |
|------|------------|------|
| **Generics** | 未使用（Go 1.26でも） | Factory Pattern・コレクション・Result型に活用 |
| **log/slog** | logrus使用 | 標準構造化ロギングとして採用 |
| **errors.Join** (Go 1.20+) | 未使用 | 複数エラーの結合に使用 |
| **google/wire** | 手動DI | コンパイルタイムDIに使用 |
| **sqlc** | Pop ORM使用 | SQLファースト型安全クエリ |
| **Atlas / goose** | Pop Migration | マイグレーション管理 |
| **go-jose/v4** | v3使用 | 最新版に統一 |
| **golangci-lint厳格設定** | 最小設定 | `gosec`, `gocritic`, `exhaustive`等を有効化 |
| **testcontainers-go** | dockertest使用 | Dockerテストの標準化 |

### 最初に定義すべきインターフェース群

Hydraの設計から抽出した、新規OAuth2プロジェクトで初日に定義すべきインターフェース:

```go
// 1. OAuth2Provider — 全エンドポイントの統合インターフェース
type OAuth2Provider interface {
    NewAuthorizeRequest(ctx context.Context, r *http.Request) (AuthorizeRequester, error)
    NewAuthorizeResponse(ctx context.Context, req AuthorizeRequester) (AuthorizeResponder, error)
    NewTokenRequest(ctx context.Context, r *http.Request) (TokenRequester, error)
    NewTokenResponse(ctx context.Context, req TokenRequester) (TokenResponder, error)
    IntrospectToken(ctx context.Context, token string) (TokenInfo, error)
    RevokeToken(ctx context.Context, token string) error
}

// 2. TokenStrategy — トークン生成戦略
type TokenStrategy interface {
    GenerateAccessToken(ctx context.Context, req TokenRequester) (token string, err error)
    GenerateRefreshToken(ctx context.Context, req TokenRequester) (token string, err error)
    ValidateAccessToken(ctx context.Context, token string) (TokenInfo, error)
}

// 3. ClientRepository — クライアント永続化
type ClientRepository interface {
    GetClient(ctx context.Context, id string) (*Client, error)
    CreateClient(ctx context.Context, c *Client) error
    UpdateClient(ctx context.Context, c *Client) error
    DeleteClient(ctx context.Context, id string) error
    AuthenticateClient(ctx context.Context, id string, secret []byte) (*Client, error)
}

// 4. TokenRepository — トークン永続化
type TokenRepository interface {
    CreateAccessToken(ctx context.Context, signature string, req TokenRequester) error
    GetAccessToken(ctx context.Context, signature string) (TokenRequester, error)
    RevokeAccessToken(ctx context.Context, signature string) error
    CreateRefreshToken(ctx context.Context, signature string, req TokenRequester) error
    GetRefreshToken(ctx context.Context, signature string) (TokenRequester, error)
    RevokeRefreshToken(ctx context.Context, signature string) error
}

// 5. ConsentStrategy — Login/Consentフロー
type ConsentStrategy interface {
    HandleAuthorization(ctx context.Context, req AuthorizeRequester) (*ConsentResult, error)
}

// 6. KeyManager — JWK管理
type KeyManager interface {
    GenerateKey(ctx context.Context, alg string) (*jose.JSONWebKey, error)
    GetSigningKey(ctx context.Context) (*jose.JSONWebKey, error)
    RotateKeys(ctx context.Context) error
}

// 7. FlowHandler — 各OAuth2フロー
type FlowHandler interface {
    CanHandle(ctx context.Context, req TokenRequester) bool
    Handle(ctx context.Context, req TokenRequester) (TokenResponder, error)
}

// 8. Cipher — 暗号化
type Cipher interface {
    Encrypt(ctx context.Context, plaintext []byte) (string, error)
    Decrypt(ctx context.Context, ciphertext string) ([]byte, error)
}
```

---

## Part 9: 総合評価サマリー

### 5段階評価

| 観点 | 評価 | コメント |
|------|------|---------|
| **設計** | ⭐⭐⭐⭐☆ (4/5) | Provider Pattern・Compose Pattern等の優れた抽象化。RegistryのService Locator化が減点 |
| **テスト** | ⭐⭐⭐⭐⭐ (5/5) | Unit/Integration/E2E/適合性テストの4層構成。HSMテスト・ベンチマークも完備 |
| **ドキュメント** | ⭐⭐⭐⭐☆ (4/5) | README・CONTRIBUTING・DEVELOPが充実。GoDocは中程度。アーキテクチャ文書は外部サイト |
| **セキュリティ** | ⭐⭐⭐⭐⭐ (5/5) | OpenID Certified、HSM対応、AEAD暗号化、5種のCVEスキャナー、定数時間比較 |
| **拡張性** | ⭐⭐⭐⭐⭐ (5/5) | Factory/Strategy/Handler Chain で高い拡張性。新フロー追加が容易 |
| **Goらしさ** | ⭐⭐⭐⭐☆ (4/5) | インターフェース設計は秀逸。logrus/pkg/errors/generics未使用が減点 |

### 最も学ぶべきトップ3の設計判断

1. **Provider Interfaceパターン**（`fosite/config.go`）
   - 設定の依存性注入を小さなインターフェースで実現
   - テスタビリティとコンテキスト対応の両立
   - OAuth2以外のプロジェクトにも普遍的に適用可能

2. **Compose/Factory Patternによるフロー合成**（`fosite/compose/`）
   - OAuth2フローのプラグイン的な構成
   - 不要なフローを除外するのが容易
   - Open-Closed Principleの実践

3. **RFC 6749準拠エラー型 + Fluent API**（`fosite/errors.go`）
   - 仕様準拠と開発者体験の両立
   - セキュリティ（Debug情報の制御）とデバッグ容易性の共存
   - i18n対応まで考慮

### 最も注意すべきトップ3のアンチパターン

1. **Service Locator化したRegistry**（`driver/registry_sql.go`）
   - 依存関係が暗黙的で、循環依存や不要な初期化が発生しやすい
   - → wireやFxによるコンパイルタイムDIで置き換え

2. **`interface{}`の多用と型安全性の欠如**（`fosite/compose/`）
   - Factory関数の引数と戻り値が`interface{}`でランタイムエラーのリスク
   - → Go genericsで型パラメータ化

3. **レガシーライブラリへの依存**（logrus, pkg/errors, go-jose/v3）
   - メンテナンス終了や古いAPI設計のライブラリに依存
   - → slog, 標準errors, go-jose/v4に移行

### フォークして使うべきか、参考にして自前実装すべきか

| 判断軸 | フォーク | 参考にして自前実装 |
|--------|---------|------------------|
| **開発速度** | 速い（既に動く） | 遅い（ゼロから） |
| **カスタマイズ性** | 低い（既存アーキテクチャに制約） | 高い（自由設計） |
| **技術的負債** | 高い（レガシー依存含む） | 低い（最新技法を採用可能） |
| **仕様準拠** | 高い（OpenID Certified） | 自前で検証必要 |
| **メンテナンスコスト** | 高い（upstream追従必要） | 中（自前管理） |

**推奨判断:**

- **フル機能のOAuth2/OIDCサーバーが必要** → Hydraをそのまま使用（フォーク不要、デプロイして利用）
- **OAuth2ライブラリとして一部機能が必要** → fositeの設計を参考に自前実装
  - fositeのインターフェース設計（Provider Pattern, Handler Chain, Token Strategy）は直接参考にすべき
  - ただし`interface{}`多用部分はgenericsで再設計
  - ストレージ層はsqlcやEntで現代的に実装
- **学習目的** → Handler Chain, Compose Pattern, Error型の設計を重点的に学ぶ

**結論:** Hydraは「参考にして自前実装」が最適。fosite（内蔵OAuth2ライブラリ）のインターフェース設計は秀逸だが、実装の細部はレガシー依存が多く、新規プロジェクトではgenerics・slog・sqlc等の現代的なGoスタックで再実装するほうが長期的にメンテナンスしやすい。RFC/OIDC仕様の実装パターンはHydra/fositeから大いに学ぶべき。
