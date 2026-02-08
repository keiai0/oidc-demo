# 実装計画

---

## 実装の大原則：標準仕様の完全遵守

このプロジェクトが実装するOIDC/OAuth 2.0は、すべての挙動が標準仕様（RFC・OpenID Foundation仕様書）で定義されている。

**「だいたい動く実装」は認証基盤として失格である。**

仕様の読み落とし・解釈ミスは直接セキュリティホールになる。以下を必須ルールとする。

### ルール1: 実装前に該当仕様を必ず読む

各フェーズ・各エンドポイントの実装を開始する前に、対応する標準仕様の該当セクションを読了すること。「たぶんこうだろう」で実装してはならない。

| 実装対象 | 必読仕様 |
|---------|---------|
| 認可エンドポイント | RFC 6749 Section 4.1, OIDC Core Section 3.1 |
| トークンエンドポイント | RFC 6749 Section 4.1.3, 5.1, 5.2 |
| PKCE | RFC 7636 全文 |
| IDトークン | OIDC Core Section 2, 3.1.3.7 |
| userinfoエンドポイント | OIDC Core Section 5 |
| JWKSエンドポイント | RFC 7517, RFC 7518 |
| Discoveryエンドポイント | OIDC Discovery 1.0 Section 4 |
| トークン失効 | RFC 7009 全文 |
| Refresh Token Rotation | RFC 9700 Section 4.14 |
| RP-Initiated Logout | RP-Initiated Logout 1.0 全文 |
| Front-Channel Logout | Front-Channel Logout 1.0 全文 |
| Back-Channel Logout | Back-Channel Logout 1.0 全文 |

### ルール2: MUST/SHOULD/MAY を区別して実装する

RFC 2119 の用語定義に従い、各要件の強度を区別する。

- **MUST / REQUIRED / SHALL**: 必ず実装する。省略はセキュリティホール
- **SHOULD / RECOMMENDED**: 正当な理由がない限り実装する。省略する場合は理由をコメントに残す
- **MAY / OPTIONAL**: 実装するかどうか設計判断する。判断結果をドキュメントに残す

### ルール3: セキュリティ要件のチェックリストを実装完了の定義とする

[03-api-design.md](./design/03-api-design.md) のセキュリティ設計チェックリストを、各エンドポイントの実装完了条件とする。チェックリストが全て埋まるまで「実装完了」としない。

### ルール4: 仕様から外れる場合は必ず記録する

仕様上 OPTIONAL な機能を実装しない場合、または仕様の解釈に判断が必要な場合は、その判断理由をコードのコメントまたは設計ドキュメントに残す。

---

## フェーズ構成

```
Phase 0: 開発環境構築
Phase 1: OPコア（認証基盤の骨格）
Phase 2: OP管理機能
Phase 3: RP（動作検証）
Phase 4: MFA（TOTP）
Phase 5: パスキー（WebAuthn）
Phase 6: SLO（シングルログアウト）
```

各フェーズは前のフェーズが完了してから開始する。フェーズ内のタスクは依存関係のないものを並行して進めてよい。

---

## Phase 0: 開発環境構築

**目標:** 全サービスがDockerで起動し、各サービスが通信できる状態にする。

### タスク

- [ ] `docker-compose.yml` の作成
  - PostgreSQL（単一インスタンス）
  - OP backend（Go）
  - OP frontend（Next.js静的 → nginx）
  - RP（Next.js）
  - profiles: `op` / `rp` / `all`
- [ ] `docker/postgres/` の作成
  - `postgresql.conf`
  - `init/01_create_schemas.sql`（`op` / `rp` スキーマ、ロール作成）
- [ ] `.env.example` の作成（全サービスの環境変数定義）
- [ ] OP backend の `go.mod` 初期化・依存パッケージ追加
- [ ] RP の `package.json` 初期化・依存パッケージ追加
- [ ] OP frontend の `package.json` 初期化・依存パッケージ追加

### 完了条件

- `docker compose --profile all up` で全サービスが起動する
- 各サービスのヘルスチェックエンドポイントが疎通する

---

## Phase 1: OPコア（認証基盤の骨格）

**目標:** Authorization Code Flow（PKCE付き）が一通り動作する状態にする。RPがOPを経由してログインできる最小構成。

**必読仕様:** RFC 6749, OIDC Core 1.0, RFC 7636, RFC 7517/7518/7519, OIDC Discovery 1.0

### 1-1. DB・基盤

- [ ] golang-migrate のセットアップ
- [ ] マイグレーションファイルの作成（`02-domain-model.md` のテーブル定義に準拠）
  - `tenants`, `clients`, `redirect_uris`, `post_logout_redirect_uris`
  - `users`, `credentials`, `password_credentials`, `password_histories`
  - `sessions`, `authorization_codes`, `access_tokens`, `refresh_tokens`, `id_tokens`
  - `sign_keys`
- [ ] GORM初期化（`database/database.go`）
- [ ] 各テーブルの `model/` struct 定義（GORMタグ付き）
- [ ] 各 `store/` GORM実装

### 1-2. JWT署名鍵管理

- [ ] RSA鍵ペア生成処理（起動時またはキー未存在時に自動生成）
- [ ] `sign_keys` テーブルへの保存（秘密鍵はファイル or 環境変数、DBには公開鍵のみ）
- [ ] JWKSエンドポイント実装（`GET /jwks`）
  - **仕様参照:** RFC 7517 Section 5（JWK Set Format）

### 1-3. Discoveryエンドポイント

- [ ] `GET /{tenant_code}/.well-known/openid-configuration` 実装
  - **仕様参照:** OIDC Discovery 1.0 Section 4
  - `issuer` のURL末尾スラッシュルールを厳守（Section 4.3）

### 1-4. 認証フロー（内部API）

- [ ] `crypto/` パスワードハッシュ実装（argon2id）
- [ ] `POST /internal/login` 実装（ID/パスワード検証 → セッション発行）
- [ ] `GET /internal/me` 実装（セッション確認）
- [ ] ログイン画面（OP frontend）の実装

### 1-5. 認可エンドポイント

- [ ] `GET /{tenant_code}/authorize` 実装
  - **仕様参照:** RFC 6749 Section 4.1.1, OIDC Core Section 3.1.2
  - `client_id` / `redirect_uri` の登録確認（前方一致不可、完全一致）
  - セッション確認（SSO）
  - 認可コード発行（`code_challenge` / `code_challenge_method` の保存）
  - `state` / `nonce` の透過的な処理

### 1-6. トークンエンドポイント

- [ ] `POST /{tenant_code}/token` 実装（`authorization_code` grant）
  - **仕様参照:** RFC 6749 Section 4.1.3, OIDC Core Section 3.1.3
  - クライアント認証（`client_secret_basic` / `client_secret_post`）
  - 認可コードの有効期限・使用済みチェック（`used_at`）
  - PKCE検証（S256のみサポート）
    - **仕様参照:** RFC 7636 Section 4.6
  - IDトークン生成（RS256署名、必須クレーム検証）
    - **仕様参照:** OIDC Core Section 2（IDトークンの必須クレーム）
  - アクセストークン生成（JWT）
  - リフレッシュトークン発行

### 1-7. userinfoエンドポイント

- [ ] `GET /{tenant_code}/userinfo` 実装
  - **仕様参照:** OIDC Core Section 5.3
  - アクセストークン検証
  - スコープに応じたクレームフィルタリング（`openid` / `profile` / `email`）

### 1-8. Refresh Token

- [ ] `POST /{tenant_code}/token` の `refresh_token` grant 実装
  - **仕様参照:** RFC 6749 Section 6, RFC 9700 Section 4.14
  - Refresh Token Rotation（使用のたびに新しいトークンを発行）
  - Reuse Detection（無効化済みトークンの再利用検知 → セッション全体を失効）

### 1-9. トークン失効エンドポイント

- [ ] `POST /{tenant_code}/revoke` 実装
  - **仕様参照:** RFC 7009 全文
  - 存在しないトークンでも `200 OK` を返す（仕様上の要件）

### 完了条件

- Authorization Code Flow（PKCE付き）が動作する
- IDトークン・アクセストークン・リフレッシュトークンが発行される
- Refresh Token Rotation + Reuse Detection が動作する
- JWKSエンドポイントで公開鍵が取得できる
- Discoveryエンドポイントが正しいメタ情報を返す

---

## Phase 2: OP管理機能

**目標:** 管理UIからテナント・クライアント（RP）・鍵の管理ができる状態にする。

### 2-1. 管理API

- [ ] テナント管理（`/management/v1/tenants`）
  - `GET` / `POST` / `GET /{id}` / `PUT /{id}`
- [ ] クライアント管理（`/management/v1/tenants/{id}/clients`）
  - CRUD + シークレットローテーション + redirect_uri 管理
- [ ] 鍵管理（`/management/v1/keys`）
  - 一覧 / ローテーション / 無効化
- [ ] インシデント対応（`/management/v1/incidents/*`）
  - 全トークン失効 / テナント全トークン失効 / ユーザー全トークン失効

### 2-2. 管理UI（OP frontend）

- [ ] テナント管理画面
- [ ] クライアント（RP）登録・編集画面
- [ ] 鍵管理画面
- [ ] インシデント対応画面

### 完了条件

- 管理UIからクライアント（RP）を登録できる
- 登録したクライアントでPhase 1の認証フローが動作する

---

## Phase 3: RP（動作検証）

**目標:** OPに対してOIDCクライアントとして動作するRPが完成し、認証フローを画面上で検証できる。

**必読仕様:** OIDC Core 1.0（クライアント側の検証要件）、RFC 9700

### 3-1. Drizzle ORM・DB

- [ ] Drizzle スキーマ定義（`src/lib/db/schema.ts`）
  - `users`（`op_sub` をキー）、`sessions`
- [ ] マイグレーション実行環境の整備

### 3-2. OIDC クライアント実装

- [ ] `openid-client` を使ったDiscovery（OPのメタ情報取得）
- [ ] `GET /api/auth/login` 実装
  - PKCE（S256）生成
  - `state` 生成・セッション保存（CSRF対策）
  - OPの認可エンドポイントへリダイレクト
- [ ] `GET /api/auth/callback` 実装
  - `state` 検証
  - 認可コードとトークン交換
  - IDトークン検証
    - **仕様参照:** OIDC Core Section 3.1.3.7（IDトークンの検証手順）
    - `iss` / `aud` / `exp` / `iat` / `nonce` / 署名の全検証
  - セッション確立
  - `op_sub` をキーにRPのDBにユーザー情報を保存
- [ ] `POST /api/auth/logout` 実装
  - RPセッション削除
  - OPのログアウトエンドポイントへリダイレクト（`id_token_hint` 付き）

### 3-3. RP画面

- [ ] ログインページ
- [ ] 認証後ダッシュボード（以下の情報を表示）
  - IDトークンのデコード結果（クレーム一覧）
  - アクセストークン
  - セッション情報
  - userinfoエンドポイントのレスポンス

### 完了条件

- RPからOPに対してログイン・ログアウトが動作する
- ダッシュボードでトークン情報・クレームを確認できる
- IDトークンの全検証項目が実装されている

---

## Phase 4: MFA（TOTP）

**目標:** TOTP（Time-based One-Time Password）による2段階認証が動作する。

**必読仕様:** RFC 6238（TOTP）, RFC 4226（HOTP）

### タスク

- [ ] `mfa_configs` / `totp_configs` テーブルのマイグレーション追加
- [ ] `crypto/totp/` 実装
  - シークレット生成・暗号化保存
  - TOTP検証（リプレイ攻撃防止: `last_used_step` チェック）
    - **仕様参照:** RFC 6238 Section 5.2
- [ ] `POST /internal/mfa/totp/setup` 実装（QRコード生成含む）
- [ ] `POST /internal/mfa/totp/verify` 実装
- [ ] 認証フロー（Phase 1）にMFAステップを追加
  - パスワード認証成功 → MFA設定済みの場合 → TOTPコード入力画面へ
- [ ] OP frontend にTOTP入力画面を追加

### 完了条件

- TOTP設定・検証が動作する
- リプレイ攻撃防止（同一stepの再利用拒否）が実装されている

---

## Phase 5: パスキー（WebAuthn）

**目標:** パスキー（FIDO2/WebAuthn）による認証が動作する。

**必読仕様:** WebAuthn Level 2（W3C）, CTAP2

### タスク

- [ ] `webauthn_credentials` テーブルのマイグレーション追加
- [ ] `crypto/webauthn/` 実装
  - 登録フロー（credential作成・公開鍵保存・sign_countの初期化）
  - 認証フロー（assertion検証・sign_countの更新）
    - **仕様参照:** WebAuthn Level 2 Section 7.2（認証アサーションの検証手順）
    - sign_count のクローン検知を実装
- [ ] 登録・認証エンドポイント実装（`/internal/mfa/webauthn/*`）
- [ ] OP frontend にパスキー登録・認証画面を追加

### 完了条件

- パスキーの登録・認証が動作する
- sign_count によるクローン検知が実装されている

---

## Phase 6: SLO（シングルログアウト）

**目標:** RP-Initiated Logout / Front-Channel Logout / Back-Channel Logout が動作する。

**必読仕様:** RP-Initiated Logout 1.0, Front-Channel Logout 1.0, Back-Channel Logout 1.0

### タスク

- [ ] `GET /{tenant_code}/logout` 実装（RP-Initiated Logout）
  - **仕様参照:** RP-Initiated Logout 1.0 全文
  - `id_token_hint` の検証（署名・`iss`・`aud`）
  - セッション・関連トークンの失効
  - `post_logout_redirect_uri` の登録確認（完全一致）
- [ ] Front-Channel Logout 実装
  - **仕様参照:** Front-Channel Logout 1.0 全文
  - ログアウト完了ページでiframeによるRP通知
  - `iss` / `sid` は両方セットで送るか送らないか（片方だけ送ることは禁止）
- [ ] Back-Channel Logout 実装
  - **仕様参照:** Back-Channel Logout 1.0 全文
  - `logout_token` の生成（必須クレーム: `iss`, `aud`, `iat`, `exp`, `jti`, `events`）
  - `nonce` を含めてはいけない（MUST NOT）
  - RP の `backchannel_logout_uri` へのPOST
  - 通知失敗をログに記録（通知失敗でもユーザーをブロックしない）
- [ ] RP側のBack-Channel Logoutエンドポイント実装
  - `logout_token` の検証
  - 対象セッションの無効化
- [ ] `POST /internal/logout` 実装（OPセッション削除）

### 完了条件

- RP起点のログアウトでOP・全RPのセッションが失効する
- Back-Channel Logout通知が動作する
- Front-Channel Logout通知が動作する
- logout_tokenの全検証項目が実装されている

---

## 実装順序の根拠

```
Phase 0 → 1 → 2 → 3 → 4 → 5 → 6

Phase 0: 環境なしには何も始まらない
Phase 1: 認証の根幹。他の全フェーズの前提
Phase 2: Phase 1のフローをUIから操作可能にする（RP登録がPhase 3の前提）
Phase 3: OPの動作をE2Eで検証できるようにする
Phase 4: Phase 1の認証フローへの追加なのでPhase 1完了後
Phase 5: Phase 4と独立しているが、認証基盤が安定してから追加
Phase 6: セッション管理が確立している（Phase 1完了）必要があり、最後
```

---

## 仕様参照リンク集

| 仕様 | URL |
|------|-----|
| OAuth 2.0 RFC 6749 | https://datatracker.ietf.org/doc/html/rfc6749 |
| OIDC Core 1.0 | https://openid.net/specs/openid-connect-core-1_0.html |
| JWT RFC 7519 | https://datatracker.ietf.org/doc/html/rfc7519 |
| JWK RFC 7517 | https://datatracker.ietf.org/doc/html/rfc7517 |
| JWA RFC 7518 | https://datatracker.ietf.org/doc/html/rfc7518 |
| PKCE RFC 7636 | https://datatracker.ietf.org/doc/html/rfc7636 |
| Token Revocation RFC 7009 | https://datatracker.ietf.org/doc/html/rfc7009 |
| OAuth 2.0 Security BCP RFC 9700 | https://datatracker.ietf.org/doc/html/rfc9700 |
| OIDC Discovery 1.0 | https://openid.net/specs/openid-connect-discovery-1_0.html |
| RP-Initiated Logout 1.0 | https://openid.net/specs/openid-connect-rpinitiated-1_0.html |
| Front-Channel Logout 1.0 | https://openid.net/specs/openid-connect-frontchannel-1_0.html |
| Back-Channel Logout 1.0 | https://openid.net/specs/openid-connect-backchannel-1_0.html |
| WebAuthn Level 2 | https://www.w3.org/TR/webauthn-2/ |
| RFC 6238 TOTP | https://datatracker.ietf.org/doc/html/rfc6238 |
