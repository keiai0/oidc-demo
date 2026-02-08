## 技術選定

### OP（OpenID Provider）

| レイヤー | 採用技術 | 理由 |
|---------|---------|------|
| バックエンド言語 | Go | 認証基盤としての実績、明示的なエラーハンドリングがOIDC仕様準拠の実装に向いている |
| OIDCライブラリ | [ory/fosite](https://github.com/ory/fosite) | OAuth 2.0 / OIDC Core / PKCE / Token Revocation / SLO をカバーする実績あるOSSライブラリ。ただしJWT周りのみ委ねる形に留め、エンドポイント骨格は自前で実装する |
| JWTライブラリ | [lestrrat-go/jwx/v3](https://github.com/lestrrat-go/jwx) | RS256 / JWK / JWKS をサポート |
| HTTPフレームワーク | Echo | マルチテナントルーティング（`/{tenant_code}/*`）の扱いやすさ |
| DB | PostgreSQL | 設計済みのスキーマがRDB向けに最適化されている |
| ORM | GORM | Goで最も採用実績のあるORM |
| マイグレーション | golang-migrate | スキーマ変更を連番SQLで厳密に管理。ロールバック対応 |
| 管理UI | Next.js（App Router）| RPと技術スタックを統一。`output: 'export'` で静的出力しnginxが配信 |

### RP（動作検証用）

| レイヤー | 採用技術 | 理由 |
|---------|---------|------|
| フレームワーク | Next.js（App Router）| Route Handlers でOIDC CallbackハンドラとDB操作をサーバーサイドで完結できる |
| OIDCクライアント | [openid-client](https://github.com/panva/openid-client) | Discovery / PKCE / Token Validation / SLO をカバー。Node.js Runtime固定で使用 |
| DB | PostgreSQL（opとは別スキーマ） | op_sub をキーとしたRP固有ユーザー属性を管理 |
| ORM | Drizzle ORM | Next.js との相性が良く、型安全 |

### インフラ・開発環境

| 用途 | 採用技術 |
|------|---------|
| コンテナ管理 | Docker Compose（profiles で起動対象を切り替え可能） |
| パッケージ管理(JS) | pnpm |

> **Redisについて:** 認可コード・セッションはPostgreSQLのテーブル（`authorization_codes`, `sessions`）で管理し、TTLは `expires_at` カラムで制御する。現時点でRedisは導入しない。レート制限等の用途が生じた時点で追加する。

---

## アーキテクチャ方針（OPバックエンド）

認証基盤では「ビジネスロジック」の大半が RFC/OIDC 仕様で決まっており、クリーンアーキテクチャの usecase 層に書く内容が仕様手順の写しになりがちである。
また、UserInfo・Revoke・Discovery・JWKS のようにロジックが薄いエンドポイントでは usecase 層が透過的なラッパーになり、形骸化する。

そのため、OIDC の各フロー（authorize, token, userinfo 等）をそのままファイル単位とし、HTTP ハンドラとフローロジックを同居させる**フロー中心アーキテクチャ**を採用する。
Go 製の主要な OIDC/OAuth2 実装として以下を比較検討し、2つのプロジェクトを参考にした。

| プロジェクト | Stars | 運営 | 特徴 |
|-------------|-------|------|------|
| [ory/hydra](https://github.com/ory/hydra) | 15,000+ | Ory 社（CNCF Incubating） | OAuth2/OIDC 認可サーバー。ログイン UI を持たず認可判断を外部に委譲する Headless 設計。ドメインごとにパッケージを分割し、handler・persistence・ロジックを同居させる |
| [dexidp/dex](https://github.com/dexidp/dex) | 9,000+ | 元 CoreOS → コミュニティ（CNCF Sandbox） | OIDC IdP / ID ブローカー。Kubernetes 認証バックエンドとして広く利用。`server/` にハンドラとロジックを同居させるフラットな構成 |
| [zitadel/oidc](https://github.com/zitadel/oidc) | 1,300+ | ZITADEL 社 | OP/RP 両方をライブラリとして提供。ZITADEL 本体の OIDC 基盤 |
| [ory/fosite](https://github.com/ory/fosite) | — | Ory 社 | hydra の内部ライブラリ。OAuth2 フレームワークとして各 grant type を細かくカスタマイズ可能 |

**hydra と dex を参考にした理由:**

- **hydra**: 「フローごとにパッケージを分け、ハンドラとロジックを同居させる」設計パターンが、OIDC 仕様駆動の本プロジェクトと合致する。CNCF Incubating・OpenID Foundation 認定済みで、設計品質の裏付けがある
- **dex**: ハンドラとロジックをフラットに同居させるシンプルな構成が、学習・検証目的の本プロジェクトの規模感に適している。Kubernetes エコシステムでの豊富な実績から信頼性も高い
- **zitadel/oidc・fosite は不採用**: zitadel/oidc はライブラリ形式のため構成の参考にはしづらく、fosite は hydra の内部実装として間接的に参考にしている

**一般的なアーキテクチャとの違い:**

Go の Web アプリケーション全般では handler / service / repository の3層分割やクリーンアーキテクチャが主流であり、フロー中心アーキテクチャは一般的ではない。
ただし認証基盤の領域では、以下の理由からフロー中心の構成が自然に採用されやすい:

- ロジックの大半が RFC/OIDC 仕様で決まっており、usecase 層が仕様手順の転記になる
- authorize・token・userinfo 等のフロー間で共有ロジックが少なく、フローごとの独立性が高い
- discovery・jwks・userinfo 等はロジックが薄く、usecase 層が透過的ラッパーになる

hydra・dex をはじめ、認証基盤の実装では「フロー = ファイル/パッケージ」とする構成が広く見られる。

```
┌──────────────────────────────────────────────┐
│  oidc/  auth/（フローパッケージ）               │
│  HTTP ハンドラ + フローロジック一体              │
│  deps.go のインターフェース経由で               │
│  store / crypto / jwt を利用                   │
└───┬─────────────┬──────────────┬─────────────┘
    │             │              │
┌───▼────┐  ┌────▼─────┐  ┌────▼────┐
│ store/ │  │ crypto/  │  │ jwt/    │
│ GORM   │  │ argon2id │  │ 署名    │
│ 実装   │  │ AES,PKCE │  │ 検証    │
└───┬────┘  └──────────┘  └─────────┘
    │
┌───▼────┐
│ model/ │
│ エンティティ・DTO（依存なし）              │
└────────┘
```

**設計判断: model/ にGORMタグを持たせる**

model/ の struct に GORM タグを直接付与する。別途 store 固有のモデルへの変換コストを避け、実装をシンプルに保つ。

**認証方式の拡張性:**
新しい認証方式（例: TOTP、パスキー）を追加する際は `crypto/` や `jwt/` に実装を追加し、フローパッケージの `deps.go` に定義したインターフェースを満たす形で注入する。

---

## ディレクトリ構成

```
oidc-demo/
│
├── op/
│   ├── backend/                              # Go（OIDCエンドポイント + 内部API）
│   │   ├── cmd/
│   │   │   └── server/
│   │   │       └── main.go                  # 起動・DIの組み立て
│   │   ├── config/
│   │   │   └── config.go                    # 環境変数読み込み（DSN, JWT鍵パス等）
│   │   ├── internal/
│   │   │   ├── model/                       # エンティティ・DTO + GORMタグ
│   │   │   │   ├── tenant.go
│   │   │   │   ├── client.go
│   │   │   │   ├── user.go
│   │   │   │   ├── credential.go
│   │   │   │   ├── session.go
│   │   │   │   ├── sign_key.go
│   │   │   │   └── ...
│   │   │   ├── oidc/                        # OIDCフロー（ハンドラ + ロジック一体）
│   │   │   │   ├── authorize.go             # GET /{tenant_code}/authorize
│   │   │   │   ├── token.go                 # POST /{tenant_code}/token（ルーティング）
│   │   │   │   ├── token_authcode.go        # Authorization Code Grant フロー
│   │   │   │   ├── token_refresh.go         # Refresh Token Grant フロー
│   │   │   │   ├── userinfo.go              # GET /{tenant_code}/userinfo
│   │   │   │   ├── revoke.go                # POST /{tenant_code}/revoke
│   │   │   │   ├── discovery.go             # GET /{tenant_code}/.well-known/openid-configuration
│   │   │   │   ├── jwks.go                  # GET /jwks
│   │   │   │   ├── deps.go                  # 依存インターフェース・関数型
│   │   │   │   └── errors.go                # OIDCエラーヘルパー
│   │   │   ├── auth/                        # 内部API（OP Frontend 向け）
│   │   │   │   ├── login.go                 # POST /internal/login
│   │   │   │   ├── me.go                    # GET /internal/me
│   │   │   │   ├── deps.go
│   │   │   │   └── errors.go
│   │   │   ├── store/                       # 永続化（GORM実装）
│   │   │   │   ├── tenant.go
│   │   │   │   ├── client.go
│   │   │   │   ├── user.go
│   │   │   │   └── ...
│   │   │   ├── crypto/                      # 暗号処理（argon2id, AES, PKCE）
│   │   │   ├── jwt/                         # JWT署名・検証・鍵管理
│   │   │   └── database/                    # GORM初期化・マイグレーション
│   │   ├── db/
│   │   │   ├── migrations/                  # golang-migrate SQLファイル
│   │   │   └── seeds/                       # 開発用テストデータ
│   │   ├── go.mod
│   │   ├── go.sum
│   │   └── Dockerfile                       # マルチステージビルド（builder / runner）
│   │
│   └── frontend/                            # ログイン画面（Next.js、静的出力）
│       ├── src/
│       │   └── app/
│       ├── next.config.ts                   # output: 'export' 設定
│       ├── package.json
│       └── Dockerfile                       # マルチステージ（node build → nginx配信）
│
├── rp/                                      # 動作検証RP（Next.js + Drizzle ORM）
│   ├── src/
│   │   ├── app/
│   │   │   ├── api/
│   │   │   │   └── auth/                    # OIDC Callback・Logout（Node.js Runtime固定）
│   │   │   └── dashboard/                   # 認証後画面（トークン情報・セッション表示）
│   │   └── lib/
│   │       └── db/
│   │           └── schema.ts                # Drizzle スキーマ定義
│   ├── drizzle/                             # Drizzle マイグレーションファイル
│   ├── drizzle.config.ts
│   ├── package.json
│   └── Dockerfile
│
├── docker/
│   └── postgres/
│       ├── postgresql.conf                  # 接続数・ログ等のPostgreSQL設定
│       └── init/
│           └── 01_create_schemas.sql        # opスキーマ・rpスキーマ・ロール作成
│                                            # ※テーブル作成はmigrations/が担う
│
├── .env.example                             # 全サービス共通の環境変数テンプレート
└── docker-compose.yml                       # profiles対応（op / rp / all）
```
