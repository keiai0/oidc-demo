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

## Phase 7: 認証基盤補完

**目標:** Phase 4（MFA）の前提となる基本機能を整備する。認証基盤として欠かせない機能群。

**備考:** Phase 7 は Phase 4 の前に実施することを推奨。Phase 4 で ACR/AMR が必要になるため。

### 7-1. パスワードリセット

API 設計（`03-api-design.md`）に定義済みだが実装計画が未設定だった機能。

- [ ] パスワードリセットトークン生成・メール送信（`POST /internal/password/reset-request`）
  - トークンは署名付き JWT または暗号学的ランダム値
  - 有効期限: 30分以内（OWASP 推奨）
  - メール送信は外部サービス連携（SendGrid, SES 等）
- [ ] パスワードリセット実行（`POST /internal/password/reset`）
  - トークン検証 → パスワード更新
  - 使用済みトークンの再利用防止
  - パスワード履歴チェック（`password_histories` テーブル活用）
- [ ] OP frontend にパスワードリセット画面を追加

### 7-2. パスワード変更（ログイン済みユーザー向け）

パスワードリセット（メール経由）とは別フロー。ログイン済みユーザーが現在のパスワードを確認した上で変更する。

- [ ] `POST /internal/password/change` 実装
  - 現在のパスワード検証 → 新パスワード設定
  - パスワード履歴チェック
  - セッションは維持（再ログイン不要）
- [ ] OP frontend にパスワード変更画面を追加

### 7-3. アカウントロックアウト / レート制限

ブルートフォース攻撃への対策。OWASP Authentication Cheat Sheet 準拠。

- [ ] ログイン失敗カウンターの実装
  - `users` テーブルに `failed_login_count` / `locked_until` カラム追加（または Redis 等で管理）
  - N回連続失敗でアカウント一時ロック（例: 5回失敗 → 15分ロック）
  - ロック中は正しいパスワードでもログイン拒否
- [ ] レート制限の実装
  - IP ベースのレート制限（`/internal/login` エンドポイント）
  - クライアントベースのレート制限（`/token` エンドポイント）
- [ ] ロック解除手段の実装
  - 時間経過による自動解除
  - 管理者による手動解除（管理API）

### 7-4. ACR / AMR クレーム

Phase 4（MFA）で RP が認証強度を判断するための前提。

- [ ] `sessions` テーブルに MFA 追跡カラム追加
  - `authenticated_methods TEXT`（JSON 配列: `["pwd"]`, `["pwd", "otp"]`）
  - `acr_claim VARCHAR(255)`（ACR 値）
  - `auth_level SMALLINT`（0: 未認証, 1: パスワード, 2: MFA）
- [ ] `IDTokenClaims` に `ACR string` / `AMR []string` を追加
- [ ] ID トークン生成時に ACR/AMR クレームを含める
- [ ] `authorize` エンドポイントで `acr_values` パラメータを抽出・検証
  - 現在のセッションの ACR が要求を満たさない場合、再認証を要求
- [ ] Discovery レスポンスに `acr_values_supported` を追加

**仕様参照:** OIDC Core 1.0 Section 2（`acr`, `amr` クレーム定義）、Section 3.1.2.1（`acr_values` パラメータ）

### 7-5. Consent（同意）画面

OIDC Core で想定される、ユーザーが RP へのスコープ付き情報提供を承認するフロー。

- [ ] Consent 管理テーブルの追加（`user_consents`）
  - `user_id`, `client_id`, `scopes`, `granted_at`
  - 同意済みスコープの永続化（毎回聞かない）
- [ ] 認可フロー内の同意チェック
  - 未同意スコープがある場合 → 同意画面へ
  - `prompt=consent` の場合 → 常に同意画面を表示
- [ ] OP frontend に Consent 画面を追加
  - クライアント名、要求スコープ、各スコープの説明を表示

### 7-6. `prompt` / `max_age` パラメータの完全対応

SSO・ステップアップ認証の基盤。

- [ ] `prompt=none` 対応
  - セッション有効 → 認可コード発行（リダイレクト）
  - セッション無効 → `error=login_required` でリダイレクト
  - 同意未取得 → `error=consent_required` でリダイレクト
- [ ] `prompt=login` 対応
  - 有効なセッションがあっても再認証を強制
- [ ] `prompt=consent` 対応
  - 同意済みでも再同意を強制
- [ ] `max_age` 検証
  - `session.created_at` と現在時刻を比較
  - 経過時間が `max_age` を超えていたら再認証を要求
  - ID トークンに `auth_time` クレームを含める（MUST when `max_age` is used）

**仕様参照:** OIDC Core 1.0 Section 3.1.2.1

### 完了条件

- パスワードリセット・変更が動作する
- ログイン失敗時にアカウントロックが発動する
- ID トークンに ACR/AMR クレームが含まれる
- `acr_values` による認証レベル要求が機能する
- Consent 画面が表示され、同意情報が永続化される
- `prompt=none/login/consent` が仕様通りに動作する
- `max_age` による再認証要求が動作する

---

## Phase 8: セキュリティ強化

**目標:** OAuth 2.0 Security BCP（RFC 9700）の推奨事項に対応し、トークンセキュリティを強化する。

**必読仕様:** RFC 9449（DPoP）, RFC 9126（PAR）, RFC 7662（Token Introspection）, RFC 9700

### 8-1. DPoP（Demonstration of Proof-of-Possession）

Bearer トークンの盗難リスクに対する送信者制約（Sender-Constrained）トークン。RFC 9700 で推奨。

- [ ] DPoP proof JWT の検証ロジック実装
  - `DPoP` ヘッダーから JWT を取得
  - JWK Thumbprint の計算・検証
  - `htm`（HTTP メソッド）/ `htu`（URL）/ `iat` / `jti` の検証
  - リプレイ攻撃防止（`jti` の一意性チェック）
- [ ] トークンエンドポイントの DPoP 対応
  - `DPoP` ヘッダー付きリクエスト → `token_type: "DPoP"` で発行
  - アクセストークンに `cnf.jkt`（JWK Thumbprint）を埋め込み
- [ ] リソースサーバー（userinfo 等）の DPoP 対応
  - `Authorization: DPoP {token}` ヘッダーの受け入れ
  - DPoP proof と `cnf.jkt` の一致検証
- [ ] Discovery レスポンスに `dpop_signing_alg_values_supported` を追加

**仕様参照:** RFC 9449 全文

### 8-2. PAR（Pushed Authorization Requests）

認可リクエストパラメータをフロントチャネルに露出させない。FAPI 2.0 では必須。

- [ ] `POST /{tenant_code}/par` エンドポイント実装
  - クライアント認証必須
  - リクエストパラメータの検証（`client_id`, `redirect_uri`, `scope` 等）
  - `request_uri`（urn:ietf:params:oauth:request_uri:...）を発行
  - 有効期限の設定（推奨: 60秒）
- [ ] `authorize` エンドポイントの `request_uri` 対応
  - `request_uri` パラメータを受け取り、事前登録済みパラメータを復元
  - `request_uri` の有効期限・使用済みチェック
- [ ] Discovery レスポンスに `pushed_authorization_request_endpoint` / `require_pushed_authorization_requests` を追加

**仕様参照:** RFC 9126 全文

### 8-3. Token Introspection

RP がアクセストークンの有効性をサーバーサイドで確認するエンドポイント。トークン失効後の即時無効化に有効。

- [ ] `POST /{tenant_code}/introspect` エンドポイント実装
  - クライアント認証必須
  - `token` / `token_type_hint` パラメータ
  - `active: true/false` を含むレスポンス
  - `active: true` の場合: `scope`, `client_id`, `username`, `exp`, `iat`, `sub` 等を返す
- [ ] Discovery レスポンスに `introspection_endpoint` を追加

**仕様参照:** RFC 7662 全文

### 8-4. Refresh Token 有効期限ポリシー

長期間放置されたトークンの自動失効。

- [ ] 絶対有効期限（Absolute Lifetime）
  - トークン発行時点から N 日で無条件失効（例: 30日）
  - `refresh_tokens.absolute_expires_at` カラム追加
- [ ] スライディング有効期限（Sliding Window）
  - 最終使用時刻から N 時間で失効（例: 24時間未使用で失効）
  - ローテーション時に新トークンの有効期限を更新
- [ ] クライアント単位のポリシー設定
  - `clients` テーブルに `refresh_token_lifetime` / `refresh_token_idle_timeout` カラム追加

### 8-5. セキュリティヘッダー / CORS

- [ ] レスポンスヘッダーの設定
  - `Strict-Transport-Security`（HSTS）
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`（SLO の iframe 通知 URI は除外）
  - `Cache-Control: no-store`（トークンレスポンス）
  - `Content-Security-Policy`
- [ ] CORS 設定
  - `/internal/*` は OP frontend のオリジンのみ許可
  - OIDC エンドポイントは RP のオリジンに応じて許可

### 8-6. セッション固定攻撃対策

- [ ] ログイン成功時にセッション ID を再生成する
  - 旧セッション → 新セッション（同一ユーザー、新しい UUID）
  - 旧セッション ID を即座に無効化

### 完了条件

- DPoP トークンの発行・検証が動作する
- PAR 経由の認可フローが動作する
- Token Introspection が動作する
- Refresh Token の有効期限ポリシーが適用される
- セキュリティヘッダーが全エンドポイントに設定される

---

## Phase 9: 運用・可観測性

**目標:** 本番運用に必要な可観測性・運用自動化を整備する。

### 9-1. ヘルスチェック / Readiness Probe

- [ ] `GET /health` 実装（Liveness: アプリケーション生存確認）
  - 常に `200 OK` を返す（DB 接続不要）
- [ ] `GET /ready` 実装（Readiness: サービス提供可能か）
  - DB 接続確認
  - 署名鍵ロード完了確認
  - 異常時は `503 Service Unavailable`

### 9-2. 監査ログ（Audit Logging）

セキュリティインシデント調査の基盤。構造化ロギング（slog）で統一。

- [ ] slog への移行（`docs/design/05-reference-patterns.md` の方針に従う）
- [ ] 認証イベントのログ出力
  - ログイン成功 / 失敗（ユーザー ID、IP、User-Agent）
  - MFA 検証成功 / 失敗
  - トークン発行 / 失効
  - セッション作成 / 破棄
  - 管理操作（クライアント登録、鍵ローテーション等）
- [ ] ログフォーマットの統一
  - JSON 形式（構造化ログ）
  - `event_type`, `user_id`, `client_id`, `ip_address`, `timestamp`, `result` 等

### 9-3. メトリクス

- [ ] Prometheus 形式のメトリクスエンドポイント（`GET /metrics`）
- [ ] 主要メトリクスの収集
  - `op_login_total{result="success|failure"}` — ログイン試行数
  - `op_token_issued_total{grant_type="..."}` — トークン発行数
  - `op_token_revoked_total` — トークン失効数
  - `op_active_sessions` — アクティブセッション数
  - `op_http_request_duration_seconds` — エンドポイント別レスポンスタイム

### 9-4. 署名鍵ローテーションの自動化

Phase 2 の鍵管理 API を基盤に、自動ローテーションのライフサイクルを追加。

- [ ] 鍵ライフサイクル管理
  - `active` → `passive`（検証専用）→ `expired`（削除）
  - 新鍵生成から旧鍵削除までの猶予期間設定（例: 旧鍵は 7 日間検証専用で保持）
- [ ] 定期実行（cron / goroutine）による自動ローテーション
  - 設定可能な間隔（例: 90日ごと）

### 完了条件

- ヘルスチェック / Readiness Probe が動作する
- 認証イベントが構造化ログとして出力される
- Prometheus メトリクスが取得できる
- 鍵ローテーションが自動で実行される

---

## Phase 10: ユーザーセルフサービス

**目標:** ユーザーが自身のアカウント・セキュリティ設定を管理できるようにする。

### 10-1. メールアドレス変更 + 再検証

- [ ] `POST /internal/email/change-request` 実装
  - 新メールアドレスに検証メールを送信
  - 検証トークンの発行
- [ ] `POST /internal/email/verify` 実装
  - トークン検証 → メールアドレス更新 → `email_verified = true`
- [ ] OP frontend にメールアドレス変更画面を追加

### 10-2. MFA の自己管理

Phase 4/5 の派生。ユーザーが自身の MFA 設定を管理する。

- [ ] TOTP の無効化・再設定
  - `DELETE /internal/mfa/totp` — TOTP 無効化（パスワード再確認必須）
  - 再設定は既存の `POST /internal/mfa/totp/setup` を再利用
- [ ] WebAuthn デバイスの一覧・追加・削除
  - `GET /internal/mfa/webauthn/credentials` — 登録済みデバイス一覧
  - `DELETE /internal/mfa/webauthn/credentials/{id}` — デバイス削除
- [ ] バックアップコード（リカバリーコード）
  - `POST /internal/mfa/backup-codes/generate` — 10個のワンタイムコード生成
  - コードは argon2id でハッシュ化して保存
  - MFA 手段を全て失った場合の最終手段

### 10-3. セッション一覧・失効（ユーザー向け）

`01-scope.md` の「追加してよいカスタム拡張」に明記済み。

- [ ] `GET /internal/sessions` 実装
  - ユーザーのアクティブセッション一覧（IP、User-Agent、作成日時、最終アクセス）
  - 現在のセッションにマーク
- [ ] `DELETE /internal/sessions/{id}` 実装
  - 指定セッションの失効（他デバイスからのログアウト）
  - 現在のセッション以外のみ削除可能
- [ ] OP frontend にセッション管理画面を追加

### 完了条件

- メールアドレスの変更・再検証が動作する
- MFA の無効化・再設定・デバイス管理が動作する
- バックアップコードの生成・使用が動作する
- セッション一覧から他デバイスのセッションを失効できる

---

## Phase 11: OIDC 仕様追加対応

**目標:** OIDC Core の OPTIONAL 機能のうち、実用性の高いものを実装する。

### 11-1. `claims` リクエストパラメータ

スコープベースより細かいクレーム要求。RP が個別のクレームを `essential` / `voluntary` で指定できる。

- [ ] `authorize` エンドポイントで `claims` パラメータをパース（JSON）
- [ ] `id_token` / `userinfo` それぞれに対する個別クレーム要求を処理
- [ ] `essential: true` のクレームが提供できない場合のエラー処理

**仕様参照:** OIDC Core 1.0 Section 5.5

### 11-2. Pairwise Subject Identifier

RP ごとに異なる `sub` を返すプライバシー保護機能。RP 間でのユーザー名寄せを防止。

- [ ] Pairwise `sub` の生成ロジック実装
  - `sub = hash(sector_identifier, local_user_id, salt)`
  - `sector_identifier` はクライアント登録時に設定
- [ ] クライアント登録に `subject_type`（`public` / `pairwise`）設定を追加
- [ ] Discovery レスポンスの `subject_types_supported` に `pairwise` を追加

**仕様参照:** OIDC Core 1.0 Section 8

### 11-3. Userinfo の署名付き JWT レスポンス

改ざん防止が必要な場合に、userinfo を署名付き JWT で返すオプション。

- [ ] `Accept: application/jwt` ヘッダー対応
- [ ] userinfo レスポンスの JWT 署名
- [ ] クライアント登録に `userinfo_signed_response_alg` 設定を追加

**仕様参照:** OIDC Core 1.0 Section 5.3.2

### 11-4. Client Credentials Grant

M2M（サーバー間）通信用。API 設計に定義済みだが実装計画が未設定だった。

- [ ] `POST /{tenant_code}/token`（`grant_type=client_credentials`）実装
  - クライアント認証必須
  - `scope` に基づくアクセストークン発行
  - ID トークンは発行しない（ユーザー認証ではないため）
  - Refresh Token は発行しない（RFC 6749 Section 4.4.3）

**仕様参照:** RFC 6749 Section 4.4

### 完了条件

- `claims` パラメータで個別クレームを要求できる
- Pairwise Subject で RP ごとに異なる `sub` が返る
- Userinfo が JWT レスポンスに対応する
- Client Credentials Grant が動作する

---

## Phase 12: エンタープライズ拡張（将来検討）

**目標:** エンタープライズ環境で求められる連携・プロビジョニング機能。スコープが大きいため、必要に応じて個別に検討する。

### 12-1. 外部 IdP 連携（Federation）

上流の OIDC Provider / SAML IdP と連携し、認証を委譲する。

- [ ] 外部 OIDC Provider との連携（Google, Azure AD 等）
  - Credential モデルの `oidc_provider` タイプを活用
  - 外部 IdP からの ID トークン検証 → ローカルセッション発行
- [ ] JIT（Just-in-Time）プロビジョニング
  - 外部 IdP で認証されたユーザーを初回ログイン時に自動作成

### 12-2. SCIM（System for Cross-domain Identity Management）

外部システムからのユーザープロビジョニング・デプロビジョニングの自動化。

- [ ] SCIM 2.0 エンドポイント実装（RFC 7644）
  - `GET /scim/v2/Users`, `POST /scim/v2/Users`, `PATCH /scim/v2/Users/{id}`, `DELETE /scim/v2/Users/{id}`
  - ユーザースキーマ（RFC 7643）に基づくリソース表現

### 12-3. Device Authorization Grant

テレビや CLI ツールなど、ブラウザが直接使えないデバイスでの認証。

- [ ] `POST /{tenant_code}/device/authorize` 実装
  - `device_code`, `user_code`, `verification_uri` を返す
- [ ] ユーザー認可画面（ブラウザで `user_code` を入力）
- [ ] デバイス側のポーリング（`grant_type=urn:ietf:params:oauth:grant-type:device_code`）

**仕様参照:** RFC 8628 全文

### 完了条件

- 外部 IdP 経由でのログインが動作する
- SCIM によるユーザー作成・更新・削除が動作する
- Device Authorization Grant が動作する

---

## 実装順序の根拠

```
Phase 0 → 1 → 2 → 3 → 7 → 4 → 5 → 6 → 8 → 9 → 10 → 11 → 12

Phase 0:  環境なしには何も始まらない
Phase 1:  認証の根幹。他の全フェーズの前提
Phase 2:  Phase 1のフローをUIから操作可能にする（RP登録がPhase 3の前提）
Phase 3:  OPの動作をE2Eで検証できるようにする
Phase 7:  Phase 4 の前提（ACR/AMR、パスワードリセット、Consent 等）
Phase 4:  Phase 1の認証フローへのMFA追加
Phase 5:  Phase 4と独立しているが、認証基盤が安定してから追加
Phase 6:  セッション管理が確立している必要があり、MFA対応後
Phase 8:  コア機能完成後のセキュリティ強化（DPoP, PAR 等）
Phase 9:  本番運用に向けた可観測性整備
Phase 10: ユーザー向けセルフサービス機能
Phase 11: OIDC OPTIONAL 機能の追加実装
Phase 12: 必要に応じてエンタープライズ拡張
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
| Token Introspection RFC 7662 | https://datatracker.ietf.org/doc/html/rfc7662 |
| OAuth 2.0 Security BCP RFC 9700 | https://datatracker.ietf.org/doc/html/rfc9700 |
| DPoP RFC 9449 | https://datatracker.ietf.org/doc/html/rfc9449 |
| PAR RFC 9126 | https://datatracker.ietf.org/doc/html/rfc9126 |
| Device Authorization Grant RFC 8628 | https://datatracker.ietf.org/doc/html/rfc8628 |
| SCIM RFC 7644 | https://datatracker.ietf.org/doc/html/rfc7644 |
| OIDC Discovery 1.0 | https://openid.net/specs/openid-connect-discovery-1_0.html |
| RP-Initiated Logout 1.0 | https://openid.net/specs/openid-connect-rpinitiated-1_0.html |
| Front-Channel Logout 1.0 | https://openid.net/specs/openid-connect-frontchannel-1_0.html |
| Back-Channel Logout 1.0 | https://openid.net/specs/openid-connect-backchannel-1_0.html |
| WebAuthn Level 2 | https://www.w3.org/TR/webauthn-2/ |
| RFC 6238 TOTP | https://datatracker.ietf.org/doc/html/rfc6238 |
