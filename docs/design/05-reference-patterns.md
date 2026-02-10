# 参考プロジェクトから採用すべきパターン

Ory Hydra・dexidp/dex の調査（[docs/research/](../research/)）から抽出した、本プロジェクトで採用すべき設計パターンと改善指針。

調査の「Adopt / Avoid」分析を本プロジェクトの文脈で再評価し、**採用済み**・**今後採用すべき**・**不採用（理由付き）** に分類した。

---

## 1. 採用済みのパターン

Phase 0〜1 の設計・実装で既に取り入れたもの。

### 1-1. フロー中心アーキテクチャ（Hydra・Dex 共通）

**出典:** Hydra — ドメインごとにパッケージ分割し handler + logic を同居。Dex — `server/` にフラットに同居。

**採用内容:** `oidc/` パッケージで HTTP ハンドラとフローロジックを同居させる構成。usecase 層が仕様手順の転記になる問題を回避。

**根拠:** [04-tech-stack.md](./04-tech-stack.md) に詳述。

### 1-2. コンストラクタインジェクション + deps.go（Hydra の改善版）

**出典:** Hydra は Service Registry（`RegistrySQL`）で全サービスを遅延初期化するが、依存関係が暗黙的になる問題がある。

**採用内容:** Hydra の Registry パターンは**不採用**。代わりに `deps.go` にローカルインターフェースを定義し、`main.go` で明示的にコンストラクタインジェクション。Go 慣習の "Accept interfaces, return structs" に準拠。

**判断理由:** 本プロジェクトの規模では明示的 DI で十分。Service Locator のアンチパターンを避ける。

### 1-3. 純粋関数の関数型注入（Hydra 参考）

**出典:** Hydra/fosite — Strategy Pattern で各種処理を差し替え可能にする設計。

**採用内容:** `VerifyPasswordFunc`, `VerifyCodeChallengeFunc`, `ComputeATHashFunc`, `SHA256HexFunc` を関数型として `deps.go` に定義し、`main.go` で注入。

**利点:** テスト時にモック関数を注入できる。インターフェースを定義するほどでもない単一関数の依存に適している。

### 1-4. AEAD 暗号化による署名鍵の保存時暗号化（Hydra 参考）

**出典:** Hydra `aead/aesgcm.go` — AES-GCM / XChaCha20Poly1305 による保存時暗号化 + 鍵ローテーション。

**採用内容:** AES-256-GCM で秘密鍵を暗号化して DB 保存。`OP_KEY_ENCRYPTION_KEY` 環境変数で暗号鍵を管理。

### 1-5. Refresh Token Rotation + Reuse Detection（Hydra・Dex 共通）

**出典:** RFC 9700 Section 4.14.2。Hydra は fosite 内で実装、Dex は `server/refreshhandlers.go` で実装。

**採用内容:** `refresh_tokens.parent_id` によるトークンチェーン + `reuse_detected_at` による再利用検知。検知時はセッション全体を失効。

### 1-6. RFC 6749 準拠エラーコード（Hydra・Dex 共通）

**出典:** Hydra `fosite/errors.go` — `RFC6749Error` 型。Dex `server/oauth2.go` — センチネルエラー。

**採用内容:** `invalid_client`, `invalid_grant`, `unsupported_grant_type` のセンチネルエラーを定義し、RFC 準拠のエラーレスポンスを返却。

### 1-7. レガシー依存の回避（Hydra・Dex の Avoid 分析）

調査で「避けるべき」と判定したレガシー依存を全て回避:

| 避けた依存 | 代替 | 出典 |
|---|---|---|
| logrus（メンテナンス終了） | Echo デフォルトロガー | Hydra Avoid #2 |
| pkg/errors（アーカイブ済み） | 標準 `errors` + `fmt.Errorf("%w")` | Hydra Avoid #3, Dex Avoid #3 |
| gorilla/mux（アーカイブ歴あり） | Echo v4 | Dex Avoid #4 |
| Pop ORM（Ory 独自フォーク） | GORM | Hydra Avoid #6 |
| go-jose/v3（旧版） | lestrrat-go/jwx v3 | Hydra Avoid #8 |
| 複数 JWT ライブラリ混在 | lestrrat-go/jwx v3 に統一 | Hydra Avoid #9 |

---

## 2. 今後採用すべきパターン

Phase 2 以降の実装で取り入れるべきもの。優先度順。

### 2-1. Conformance Test（Dex — 最重要）

**出典:** Dex `storage/conformance/conformance.go` — 全ストレージバックエンドに共通のテストスイートを提供。

**パターン:**

```go
// store/conformance/conformance.go
func RunTests(t *testing.T, newStore func(t *testing.T) store.Store) {
    t.Run("AuthorizationCodeCRUD", func(t *testing.T) {
        s := newStore(t)
        // Create → Find → MarkAsUsed → Find(used) の一連を検証
    })
    t.Run("RefreshTokenCRUD", func(t *testing.T) { ... })
    t.Run("SessionCRUD", func(t *testing.T) { ... })
    // ...
}

// store/postgres/postgres_test.go
func TestPostgresStore(t *testing.T) {
    conformance.RunTests(t, func(t *testing.T) store.Store {
        return postgres.New(testDSN)
    })
}
```

**なぜ採用すべきか:**
- インターフェースの契約をテストとして形式化する
- 新しいストレージバックエンド追加時に `conformance.RunTests()` を呼ぶだけで正しさを保証
- 本プロジェクトの `store/` パッケージは現在 9 リポジトリに分かれており、統一的なテストスイートの価値が高い

**導入タイミング:** Phase 2 開始前。テスト基盤として先に整備する。

### 2-2. コンパイル時インターフェースチェック（Dex 参考）

**出典:** Dex — 各実装ファイルで `var _ Interface = (*Impl)(nil)` を宣言。

**パターン:**

```go
// store/ の各ファイル末尾に追加
var _ oidc.ClientFinder = (*ClientRepository)(nil)
var _ oidc.AuthorizationCodeStore = (*AuthorizationCodeRepository)(nil)
var _ oidc.AccessTokenStore = (*AccessTokenRepository)(nil)
var _ oidc.RefreshTokenStore = (*RefreshTokenRepository)(nil)
var _ auth.SessionStore = (*SessionRepository)(nil)
var _ jwt.SignKeyRepository = (*SignKeyRepository)(nil)
```

**なぜ採用すべきか:**
- インターフェースのメソッドシグネチャ変更時にコンパイルエラーで即座に検出
- 「どの構造体がどのインターフェースを満たすか」のドキュメント効果
- コストゼロ（ランタイムへの影響なし）

**導入タイミング:** 即座に導入可能。既存コードへの変更なし。

### 2-3. 構造化 OIDC エラー型（Hydra 参考）

**出典:** Hydra `fosite/errors.go` — `RFC6749Error` 型 + Fluent API。

**パターン:**

```go
// oidc/errors.go に追加
type OIDCError struct {
    Code        string // OAuth2 エラーコード (invalid_request 等)
    Description string // ユーザー向け説明
    Hint        string // 開発者向けヒント（ログ用、レスポンスには含めない）
    StatusCode  int    // HTTP ステータスコード
    Cause       error  // 元エラー
}

func (e *OIDCError) Error() string { return e.Code + ": " + e.Description }
func (e *OIDCError) Unwrap() error { return e.Cause }

func (e *OIDCError) WithDescription(desc string) *OIDCError {
    clone := *e
    clone.Description = desc
    return &clone
}

func (e *OIDCError) WithHint(hint string) *OIDCError {
    clone := *e
    clone.Hint = hint
    return &clone
}

func (e *OIDCError) WithCause(err error) *OIDCError {
    clone := *e
    clone.Cause = err
    return &clone
}

// 定義済みエラー
var (
    ErrInvalidRequest = &OIDCError{Code: "invalid_request", StatusCode: 400}
    ErrInvalidClient  = &OIDCError{Code: "invalid_client", StatusCode: 401}
    ErrInvalidGrant   = &OIDCError{Code: "invalid_grant", StatusCode: 400}
    ErrInvalidScope   = &OIDCError{Code: "invalid_scope", StatusCode: 400}
    // ...
)

// 使用例
return ErrInvalidGrant.
    WithDescription("The authorization code has already been used").
    WithHint("code was used at " + code.UsedAt.String()).
    WithCause(err)
```

**なぜ採用すべきか:**
- エンドポイント増加に伴い、エラーレスポンスの構築が散在するのを防ぐ
- `Hint` フィールドでログには詳細を残しつつレスポンスには含めない（セキュリティ）
- `errors.Is` / `errors.As` と互換（`Unwrap()` 実装）
- RFC 6749 Section 5.2 のエラーレスポンス形式に一元的に対応

**導入タイミング:** Phase 2（管理 API 追加時）。現状のセンチネルエラーからの移行。

### 2-4. time 関数の注入（Dex 参考）

**出典:** Dex `server/server.go:112` — `Now func() time.Time` を Config 経由で渡す。

**パターン:**

```go
// config または各サービスの引数で注入
type TokenServiceConfig struct {
    Now func() time.Time // nil なら time.Now
}

func (s *TokenService) now() time.Time {
    if s.config.Now != nil {
        return s.config.Now()
    }
    return time.Now()
}
```

**なぜ採用すべきか:**
- トークン有効期限・セッション有効性・鍵ローテーションのテストが決定論的に書ける
- OAuth2/OIDC は時間に強く依存するドメインであり、テスタビリティへの効果が大きい

**導入タイミング:** テスト拡充時。`jwt/`, `oidc/`, `auth/` の各サービスに段階的に導入。

### 2-5. Updater 関数パターン（Dex 参考）

**出典:** Dex `storage/storage.go:117-138` — `UpdateClient(ctx, id, func(old Client) (Client, error)) error`

**パターン:**

```go
// store のインターフェース
UpdateRefreshToken(ctx context.Context, id uuid.UUID,
    updater func(old model.RefreshToken) (model.RefreshToken, error)) error

// 実装側（GORM）
func (r *RefreshTokenRepository) UpdateRefreshToken(ctx context.Context, id uuid.UUID,
    updater func(old model.RefreshToken) (model.RefreshToken, error)) error {
    return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        var token model.RefreshToken
        if err := tx.First(&token, "id = ?", id).Error; err != nil {
            return err
        }
        updated, err := updater(token)
        if err != nil {
            return err
        }
        return tx.Save(&updated).Error
    })
}
```

**なぜ採用すべきか:**
- read-modify-write をトランザクション内で安全に実行
- 呼び出し側はトランザクション管理を意識せずロジックだけに集中できる
- Refresh Token Rotation のような「読んで → 状態を変えて → 書く」操作に最適

**導入タイミング:** store リファクタリング時。特に Refresh Token Rotation の実装を改善する際。

### 2-6. 構造化ロギング（slog）

**出典:** Dex は `log/slog` を採用（Go 1.21+ 標準）。Hydra は logrus（レガシー）。

**パターン:**

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

// リクエストコンテキスト付きロギング
logger.InfoContext(ctx, "token issued",
    slog.String("client_id", clientID),
    slog.String("grant_type", "authorization_code"),
    slog.String("scope", scope),
)
```

**なぜ採用すべきか:**
- Go 標準ライブラリ（外部依存なし）
- 構造化ログで運用時のフィルタリング・集約が容易
- Echo のロガーと併用可能

**導入タイミング:** Phase 2 以降。Echo ミドルウェアとの統合を含めて設計。

---

## 3. 不採用のパターン（理由付き）

調査で有用と判定したが、本プロジェクトの文脈では採用しないもの。

### 3-1. Handler Chain / Compose Pattern（Hydra / fosite）

**出典:** Hydra `fosite/compose/compose.go` — Factory 関数群を合成して OAuth2Provider を構築。

**不採用理由:** 本プロジェクトは学習・検証目的であり、フロー（Authorization Code, Refresh Token）が少数で固定。fosite のように grant type をプラグイン的に追加・除外する必要性がない。過剰な抽象化になる。

### 3-2. Provider Interface による設定の DI（Hydra / fosite）

**出典:** Hydra `fosite/config.go` — 設定値ごとに小さな Provider インターフェースを定義。

```go
type AccessTokenLifespanProvider interface {
    GetAccessTokenLifespan(ctx context.Context) time.Duration
}
```

**不採用理由:** 本プロジェクトではテナント設定を DB から取得する構成で、リクエストスコープの設定解決は `tenant.AccessTokenLifetime` で直接参照している。マルチテナント対応は DB クエリで完結しており、インターフェース分離の利点が薄い。

### 3-3. State Machine Pattern（Hydra）

**出典:** Hydra `flow/flow.go` — Login → Consent → Token 発行を状態機械で管理。

**不採用理由:** Hydra は Login/Consent を外部アプリに委譲する Headless 設計のため、複数ステップの状態管理が必要。本プロジェクトはログイン画面を OP 自身が提供するため、状態機械を導入する複雑さに見合わない。

### 3-4. NID（Network ID）マルチテナント（Hydra）

**出典:** Hydra `persistence/sql/persister.go` — 全テーブルに NID カラムで隔離。

**不採用理由:** 本プロジェクトは `tenant_id` による FK ベースのマルチテナントを採用済み。NID パターンは SaaS 基盤での完全隔離向けで、本プロジェクトの規模には過剰。

### 3-5. Static Storage Decorator（Dex）

**出典:** Dex `storage/static.go` — 設定ファイルの静的値を動的ストレージの上にオーバーレイ。

**不採用理由:** 本プロジェクトはクライアント・テナントを全て DB 管理。設定ファイルベースの管理は不要。

### 3-6. google/wire によるコンパイルタイム DI（Hydra 分析の代替案）

**出典:** Hydra 分析 Part 7 で Registry の代替として推奨。

**不採用理由:** 現状の `main.go` での手動 DI で依存グラフが十分に管理可能（53 ファイル規模）。wire の導入はコード生成ステップを増やすデメリットのほうが大きい。

---

## 4. 技術スタック選定への影響

調査結果を踏まえた技術選定の裏付け。[04-tech-stack.md](./04-tech-stack.md) の選定理由を補強する。

| 選定項目 | 本プロジェクト | Hydra | Dex | 選定根拠 |
|---|---|---|---|---|
| HTTP フレームワーク | Echo v4 | negroni（薄い） | gorilla/mux | Echo はルーティング・ミドルウェアが充実。gorilla/mux のアーカイブ歴を考慮 |
| ORM | GORM | Pop（Ory 独自） | Ent（実験的） | GORM はエコシステム最大。Pop は独自フォーク、Ent は実験的 |
| JWT | lestrrat-go/jwx v3 | go-jose/v3 + golang-jwt/v5 + 独自 | go-jose/v4 | jwx v3 は JWK/JWS/JWT を統一的にカバー。複数ライブラリ混在を回避 |
| マイグレーション | golang-migrate | Pop Migration | Ent + goose | golang-migrate は SQL ファースト。Pop/Ent は ORM 依存 |
| ログ | Echo デフォルト → slog 移行予定 | logrus（レガシー） | slog（標準） | Dex と同じく slog を最終目標とする |
| エラーラッピング | 標準 errors | pkg/errors | pkg/errors | 標準のみ使用。pkg/errors はアーカイブ済み |

---

## 5. 参照元ドキュメント

| ドキュメント | 内容 |
|---|---|
| [docs/research/hydra-analysis.md](../research/hydra-analysis.md) | Ory Hydra 徹底分析（9 Part 構成） |
| [docs/research/dex-analysis.md](../research/dex-analysis.md) | dexidp/dex 徹底分析（9 Part 構成） |
