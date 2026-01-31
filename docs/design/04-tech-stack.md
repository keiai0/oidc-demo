## 技術選定

### OP（OpenID Provider）

| レイヤー | 採用技術 | 理由 |
|---------|---------|------|
| バックエンド言語 | Go | 認証基盤としての実績、明示的なエラーハンドリングがOIDC仕様準拠の実装に向いている |
| OIDCライブラリ | [ory/fosite](https://github.com/ory/fosite) | OAuth 2.0 / OIDC Core / PKCE / Token Revocation / SLO をカバーする実績あるOSSライブラリ。ただしJWT周りのみ委ねる形に留め、エンドポイント骨格は自前で実装する |
| JWTライブラリ | [lestrrat-go/jwx/v2](https://github.com/lestrrat-go/jwx) | RS256 / JWK / JWKS をサポート |
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

パスワード認証・TOTP・パスキー（WebAuthn）等の複数認証方式を拡張しやすくするため、クリーンアーキテクチャで構成する。

```
┌─────────────────────────────────────────────┐
│  handler/（Delivery層）                      │
│  Echo ハンドラ。HTTPの入出力のみ担当          │
│  OIDCエンドポイント / 管理API / 内部API       │
└────────────────────┬────────────────────────┘
                     │ 呼び出し
┌────────────────────▼────────────────────────┐
│  usecase/（Usecase層）                       │
│  認証フロー・トークン発行・MFA検証等の         │
│  アプリケーションロジック                     │
│  port/ のinterfaceに依存し実装詳細を知らない  │
└──────┬─────────────────────┬────────────────┘
       │                     │
┌──────▼──────┐     ┌────────▼───────────────┐
│ domain/     │     │ port/（Port層）          │
│ エンティティ  │     │ Repository / Service の │
│ + GORMタグ  │     │ interfaceのみ定義        │
└─────────────┘     └────────┬───────────────┘
                              │ 実装
                     ┌────────▼───────────────┐
                     │ infrastructure/         │
                     │ ・database/  GORM初期化 │
                     │ ・repository/ GORM実装  │
                     │ ・crypto/  argon2id等   │
                     │ ・jwt/     署名・検証    │
                     │ ・totp/    TOTP処理     │
                     │ ・webauthn/ パスキー処理 │
                     └────────────────────────┘
```

**設計判断: domain/ にGORMタグを持たせる**

厳密なクリーンアーキテクチャでは `domain/` のstructにORMタグを付けず、`infrastructure/repository/` に変換用structを別定義する。ただしこのプロジェクトでは実装のシンプルさを優先し、`domain/` のstructにGORMタグを直接付与する方針を採用する。

**認証方式の拡張性:**
新しい認証方式（例: SMS OTP、マジックリンク）を追加する際は `infrastructure/` に実装を追加し、`port/service/` のinterfaceを満たすだけでよい。usecase層・handler層への変更を最小化できる。

---

## ディレクトリ構成

```
oidc-demo/
│
├── op/
│   ├── backend/                              # Go（OIDCエンドポイント + 管理API）
│   │   ├── cmd/
│   │   │   └── server/
│   │   │       └── main.go                  # 起動・DIの組み立て
│   │   ├── config/
│   │   │   └── config.go                    # 環境変数読み込み（DSN, JWT鍵パス等）
│   │   ├── internal/
│   │   │   ├── handler/                     # Delivery層（Echoハンドラ）
│   │   │   │   ├── oidc/
│   │   │   │   │   ├── authorize/           # GET /{tenant_code}/authorize
│   │   │   │   │   ├── token/               # POST /{tenant_code}/token
│   │   │   │   │   ├── userinfo/            # GET /{tenant_code}/userinfo
│   │   │   │   │   ├── revoke/              # POST /{tenant_code}/revoke
│   │   │   │   │   ├── logout/              # GET|POST /{tenant_code}/logout
│   │   │   │   │   └── discovery/           # /.well-known/openid-configuration, /jwks
│   │   │   │   ├── management/              # /management/v1/*
│   │   │   │   └── internal/                # /internal/*（ログイン画面向け内部API）
│   │   │   ├── usecase/                     # Usecase層
│   │   │   │   ├── auth/                    # パスワード/TOTP/パスキー認証フロー
│   │   │   │   ├── token/                   # トークン発行・検証・失効・ローテーション
│   │   │   │   ├── session/                 # セッション管理・SLO
│   │   │   │   └── management/              # テナント・クライアント・鍵管理
│   │   │   ├── domain/                      # エンティティ + GORMタグ
│   │   │   │   ├── tenant.go
│   │   │   │   ├── client.go
│   │   │   │   ├── user.go
│   │   │   │   ├── credential.go
│   │   │   │   ├── mfa.go
│   │   │   │   ├── session.go
│   │   │   │   ├── token.go
│   │   │   │   └── sign_key.go
│   │   │   ├── port/                        # Port層（interfaceの定義のみ）
│   │   │   │   ├── repository/              # DB操作のinterface
│   │   │   │   └── service/                 # JWT・crypto・TOTP・WebAuthnのinterface
│   │   │   └── infrastructure/              # Adapter層（実装）
│   │   │       ├── database/
│   │   │       │   └── database.go          # GORM初期化・接続プール・DSN設定
│   │   │       ├── repository/              # port/repository の実装（GORM使用）
│   │   │       ├── crypto/                  # argon2id によるパスワードハッシュ
│   │   │       ├── jwt/                     # lestrrat-go/jwx による署名・検証
│   │   │       ├── totp/                    # TOTP生成・検証
│   │   │       └── webauthn/                # パスキー登録・認証
│   │   ├── migrations/                      # golang-migrate SQLファイル
│   │   │   ├── 000001_create_tenants.up.sql
│   │   │   ├── 000001_create_tenants.down.sql
│   │   │   └── ...
│   │   ├── go.mod
│   │   ├── go.sum
│   │   └── Dockerfile                       # マルチステージビルド（builder / runner）
│   │
│   └── frontend/                            # 管理UI（Next.js、静的出力）
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
