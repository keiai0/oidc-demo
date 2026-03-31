# 拡張機能 実装計画

## Context

Phase 0〜11 + セキュリティデモまで実装済み。OIDC Core / OAuth 2.0 の主要仕様は網羅されている状態。

本計画では、認証基盤の理解をさらに深めるための拡張機能を追加する。

---

## 対象機能と優先順位

| 優先度 | Phase | 機能 | 必読仕様 |
|--------|-------|------|----------|
| 高 | Phase 13 | Conformance Test Suite | OIDC Core 1.0, RFC 6749, RFC 7636 |
| 高 | Phase 14 | Token Exchange | RFC 8693 |
| 中 | Phase 15 | Dynamic Client Registration | RFC 7591 / RFC 7592 |
| 中 | Phase 16 | Rich Authorization Requests (RAR) | RFC 9396 |

---

## Phase 13: Conformance Test Suite

**目標:** OIDC Core / OAuth 2.0 の主要フローが仕様通りに動作することを、自動テストで保証する。テストを書く過程で仕様の読み落としを発見・修正する。

**必読仕様:** OIDC Core 1.0 Section 3（Authentication）, RFC 6749 Section 4/5, RFC 7636, OIDC Discovery 1.0

### 設計方針

1. **Go の標準テストフレームワーク**（`testing` パッケージ）で実装する
2. **テスト対象はHTTPレベル**（エンドポイントに対するリクエスト・レスポンスの検証）
3. **テストカテゴリは OpenID Foundation の認定テストプロファイルに準拠**する
4. **テストヘルパーを `test/` ディレクトリに共通化**する
5. **実際の DB を使うインテグレーションテスト**とする（モック不使用）

### 13-1. テスト基盤の構築

- [x] `test/` ディレクトリの作成（テストヘルパー・フィクスチャ）
- [x] テスト用 DB セットアップヘルパー
  - テスト用テナント・クライアント・ユーザーの自動作成
  - テスト間の分離（トランザクションロールバック or テーブルクリーンアップ）
- [x] HTTP テストクライアントヘルパー
  - Echo の `httptest` を使ったリクエスト送信
  - レスポンスのJSON/JWT パース・検証ユーティリティ
- [x] OIDC フロー実行ヘルパー
  - Authorization Code Flow の一連のステップを自動実行する関数
  - PKCE（S256）の生成・検証
  - state / nonce の生成・検証

### 13-2. Discovery テスト（Config OP プロファイル）

> OpenID Foundation 認定の「Config OP」プロファイルに相当

- [x] `/.well-known/openid-configuration` のレスポンス検証
  - 必須フィールドの存在確認（`issuer`, `authorization_endpoint`, `token_endpoint`, `jwks_uri`, `response_types_supported`, `subject_types_supported`, `id_token_signing_alg_values_supported`）
  - `issuer` の末尾スラッシュルール（OIDC Discovery Section 4.3）
  - 各エンドポイント URL の疎通確認
- [x] JWKS エンドポイントの検証
  - JWK Set のフォーマット検証（RFC 7517 Section 5）
  - `kid`, `kty`, `alg`, `use` フィールドの存在確認
  - 公開鍵でダミー JWT の署名検証が可能であること

### 13-3. Authorization Code Flow テスト（Basic OP プロファイル）

> OpenID Foundation 認定の「Basic OP」プロファイルに相当

- [x] 正常系: Authorization Code Flow + PKCE
  - 認可リクエスト → ログイン → 認可コード発行 → トークン交換 → ID Token 検証
  - PKCE（S256）の code_verifier / code_challenge の検証
  - state パラメータの透過・検証
  - nonce クレームの ID Token 内存在確認
- [x] ID Token の検証項目（OIDC Core Section 3.1.3.7）
  - `iss` がディスカバリの `issuer` と一致すること
  - `aud` に `client_id` が含まれること
  - `exp` が未来であること
  - `iat` が妥当な範囲であること
  - `nonce` がリクエスト時の値と一致すること
  - RS256 署名が JWKS の公開鍵で検証できること
  - `at_hash` が正しいこと（OIDC Core Section 3.1.3.6）
- [x] 異常系テスト
  - 無効な `client_id` → エラーレスポンス
  - 未登録の `redirect_uri` → エラーレスポンス（リダイレクトしない）
  - PKCE `code_challenge_method=plain` → 拒否
  - `code_verifier` 不一致 → `invalid_grant`
  - 期限切れ認可コード → `invalid_grant`
  - 認可コード再利用 → `invalid_grant` + 関連トークン失効
  - `state` 不一致（RP 側検証）

### 13-4. Token Endpoint テスト

- [x] クライアント認証
  - `client_secret_basic` による認証
  - `client_secret_post` による認証
  - 不正な `client_secret` → `invalid_client`
- [x] Refresh Token Grant
  - 正常な refresh → 新しい access_token + refresh_token
  - Rotation: 旧 refresh_token が無効化されること
  - Reuse Detection: 無効化済み refresh_token の使用 → セッション全体失効（RFC 9700）
- [x] Client Credentials Grant
  - 正常なトークン発行
  - scope の制限が正しく動作すること
- [x] エラーレスポンスフォーマット（RFC 6749 Section 5.2）
  - `error`, `error_description` フィールドの存在
  - HTTP ステータスコードの正しさ

### 13-5. UserInfo テスト

- [x] 正常系
  - 有効なアクセストークンで userinfo 取得
  - `sub` が ID Token の `sub` と一致すること
  - scope に応じたクレームフィルタリング（`profile`, `email`）
- [x] 異常系
  - 無効なトークン → `401 Unauthorized`
  - 期限切れトークン → `401 Unauthorized`

### 13-6. Revocation / Introspection テスト

- [x] Token Revocation（RFC 7009）
  - access_token の失効 → introspection で `active: false`
  - refresh_token の失効 → 関連 access_token も失効
  - 存在しないトークンでも `200 OK`（RFC 7009 要件）
- [x] Token Introspection（RFC 7662）
  - 有効なトークン → `active: true` + クレーム
  - 無効なトークン → `active: false`

### 13-7. PAR テスト（RFC 9126）

- [x] 正常系
  - PAR → `request_uri` 取得 → 認可リクエストに使用 → トークン交換
  - `request_uri` の有効期限検証
- [x] 異常系
  - `request_uri` の再利用 → 拒否
  - 期限切れ `request_uri` → 拒否

### 13-8. ログアウトテスト

- [x] RP-Initiated Logout
  - `id_token_hint` 付きログアウト → セッション失効
  - `post_logout_redirect_uri` の検証
- [ ] Back-Channel Logout
  - ログアウト時に `backchannel_logout_uri` へ通知が送信されること
  - `logout_token` の検証（`iss`, `aud`, `iat`, `exp`, `jti`, `events`, `sid`）
  - `logout_token` に `nonce` が含まれないこと（MUST NOT）

### 完了条件

- [x] 全テストカテゴリでテストが pass する
- [x] テスト実行で発見された仕様違反が修正されている
- [x] `cd op/backend && go test ./test/...` で全テストが実行できる

---

## Phase 14: Token Exchange（RFC 8693）

**目標:** OAuth 2.0 Token Exchange を実装し、マイクロサービス間でのトークン委譲（delegation / impersonation）を理解する。

**必読仕様:** RFC 8693 全文

### 学習ポイント

- **Impersonation vs Delegation** の違い
  - Impersonation: 発行されるトークンの `sub` は元のユーザー（actor は見えない）
  - Delegation: `act` クレームで委譲チェーンを明示する
- **`subject_token` / `actor_token`** の役割
- **`audience` / `resource`** によるトークンのスコープ制御
- 既存の token endpoint に新しい grant_type を追加するパターン

### 14-1. データモデル

- [x] `token_exchange_policies` テーブルの作成
  - どのクライアントがどのトークンタイプの交換を許可されるかのポリシー
  - `id`, `client_id`, `allowed_subject_token_types`, `allowed_requested_token_types`, `allowed_audiences`, `allow_impersonation`, `allow_delegation`
  - **仕様参照:** RFC 8693 Section 1（セキュリティ上、AS はポリシーに基づいて交換を制御する SHOULD）

### 14-2. Token Exchange Grant 実装

- [x] `internal/oidc/token_exchange.go` の作成
  - grant_type: `urn:ietf:params:oauth:grant-type:token-exchange`
  - **仕様参照:** RFC 8693 Section 2.1（リクエストパラメータ）
- [x] リクエストパラメータの検証
  - `subject_token`（REQUIRED）: 対象者のトークン
  - `subject_token_type`（REQUIRED）: トークンタイプ URI
  - `actor_token`（OPTIONAL）: 委任者のトークン
  - `actor_token_type`（actor_token がある場合 REQUIRED）
  - `resource`（OPTIONAL）: ターゲットリソース URI
  - `audience`（OPTIONAL）: ターゲットサービスの論理名
  - `scope`（OPTIONAL）: 要求するスコープ
  - `requested_token_type`（OPTIONAL）: 要求するトークンタイプ
- [x] Subject Token の検証
  - `urn:ietf:params:oauth:token-type:access_token` → アクセストークンとして検証
  - `urn:ietf:params:oauth:token-type:id_token` → ID Token として検証
  - `urn:ietf:params:oauth:token-type:jwt` → JWT として署名検証
- [x] ポリシーチェック
  - クライアントが token exchange を許可されているか
  - 要求されたトークンタイプが許可されているか
  - audience が許可されているか

### 14-3. Impersonation（なりすまし）

- [x] Impersonation トークンの発行
  - `sub` は subject_token のユーザー
  - `client_id` は交換を要求したクライアント
  - `scope` は要求されたスコープ（元のスコープ以下に制限）
  - **仕様参照:** RFC 8693 Section 2.1（Impersonation Semantics）
- [x] レスポンス
  - `issued_token_type`: 発行されたトークンのタイプ URI
  - `token_type`: `Bearer`
  - `expires_in`: 有効期限（元のトークンより短くする SHOULD）

### 14-4. Delegation（委任）

- [x] `act` クレームの実装
  - **仕様参照:** RFC 8693 Section 4.1（`act` Claim）
  - `act` は委任チェーンを表すネストされたオブジェクト
  - 例: `{"act": {"sub": "service-a", "act": {"sub": "service-b"}}}`
- [x] `may_act` クレームの検証
  - **仕様参照:** RFC 8693 Section 4.2（`may_act` Claim）
  - subject_token に `may_act` がある場合、actor がその条件を満たすか検証
- [x] Delegation トークンの発行
  - `sub` は元のユーザー
  - `act` クレームで委任チェーンを記録

### 14-5. token endpoint への統合

- [x] `internal/oidc/token.go` に token-exchange ディスパッチを追加
- [x] Discovery エンドポイントの更新
  - `grant_types_supported` に `urn:ietf:params:oauth:grant-type:token-exchange` を追加
- [x] エラーハンドリング
  - `invalid_request`: 必須パラメータ不足
  - `invalid_grant`: subject_token / actor_token の検証失敗
  - `invalid_target`: audience / resource が許可されていない（RFC 8693 Section 2.2.1）

### 14-6. 管理 API・UI

- [x] Token Exchange ポリシーの CRUD API
  - `GET /management/v1/clients/:id/token-exchange-policy`
  - `PUT /management/v1/clients/:id/token-exchange-policy`
- [x] OP Frontend にポリシー設定画面を追加

### 14-7. RP での動作検証 UI

- [x] RP に Token Exchange デモページを追加
  - 取得済みアクセストークンを subject_token として交換リクエストを送信
  - Impersonation / Delegation の切り替え
  - 交換前後のトークンのクレーム比較表示
  - `act` クレームの委任チェーン可視化

### 完了条件

- [x] Impersonation でトークン交換が動作する
- [x] Delegation で `act` クレーム付きトークンが発行される
- [x] ポリシー未設定のクライアントからの交換が拒否される
- [x] RP のデモ UI で交換前後のトークンを比較確認できる
- [x] `cd op/backend && go test ./...` が pass する

---

## Phase 15: Dynamic Client Registration（RFC 7591 / 7592）

**目標:** OIDC クライアントが自動でOP に登録・更新・削除できる仕組みを実装する。Management API との役割の違い、ソフトウェアステートメントの検証を理解する。

**必読仕様:** RFC 7591 全文, RFC 7592 全文, OIDC Dynamic Registration 1.0

### 学習ポイント

- **Management API との違い**: Management API は管理者向け（内部）、Dynamic Registration はクライアント自身が自律的に登録する（外部）
- **Software Statement**: クライアントのメタデータを署名付き JWT で検証するパターン
- **Registration Access Token**: 登録後のクライアント管理に使う専用トークン
- **セキュリティ**: 無制限な登録は DoS やスパムの温床になるため、Initial Access Token での制御が重要

### 15-1. データモデル

- [ ] `client_registrations` テーブルの作成
  - `id`, `client_id`（FK）, `registration_access_token_hash`, `registration_client_uri`
  - `software_id`, `software_version`, `software_statement`
  - `initial_access_token_id`（どの IAT で登録されたか追跡）
  - `created_at`, `updated_at`
- [ ] `initial_access_tokens` テーブルの作成
  - 管理者が発行する、登録を許可するためのトークン
  - `id`, `token_hash`, `tenant_id`, `max_registrations`, `used_count`, `expires_at`

### 15-2. Registration Endpoint（RFC 7591）

- [ ] `POST /{tenant_code}/register` の実装
  - **仕様参照:** RFC 7591 Section 3.1（Client Registration Request）
- [ ] リクエストパラメータの処理
  - `redirect_uris`: リダイレクト URI の配列
  - `token_endpoint_auth_method`: クライアント認証方式（デフォルト: `client_secret_basic`）
  - `grant_types`: 許可する grant type の配列
  - `response_types`: 許可する response type の配列
  - `client_name`: クライアント名
  - `client_uri`: クライアント情報 URL
  - `logo_uri`: ロゴ URL
  - `scope`: 要求するスコープ
  - `contacts`: 連絡先メールアドレスの配列
  - `tos_uri`: 利用規約 URL
  - `policy_uri`: プライバシーポリシー URL
  - `software_id`: ソフトウェア識別子
  - `software_version`: バージョン文字列
  - `software_statement`: ソフトウェアステートメント JWT
- [ ] メタデータバリデーション
  - `redirect_uris` の形式検証（HTTPS 必須、localhost 例外）
  - `grant_types` と `response_types` の整合性チェック
  - `token_endpoint_auth_method` がサポートされているか
- [ ] Software Statement の検証（OPTIONAL だが実装する）
  - JWT 署名の検証（信頼する発行者の公開鍵で）
  - `software_id` の一致確認
  - ステートメント内のメタデータとリクエストの整合性
- [ ] レスポンス（HTTP 201 Created）
  - `client_id`: 発行されたクライアント ID
  - `client_secret`: 発行されたクライアントシークレット（confidential client の場合）
  - `client_id_issued_at`: 発行日時（UNIX タイムスタンプ）
  - `client_secret_expires_at`: シークレット有効期限（0 = 無期限）
  - `registration_access_token`: 管理用トークン
  - `registration_client_uri`: クライアント設定エンドポイント URL
  - 登録されたメタデータの全フィールド
- [ ] Initial Access Token による認可
  - Bearer トークンが必要（管理者が事前発行）
  - トークンの有効期限・使用回数の検証
- [ ] エラーレスポンス
  - `invalid_redirect_uri`: 無効なリダイレクト URI
  - `invalid_client_metadata`: 不正なメタデータ
  - `invalid_software_statement`: ソフトウェアステートメント検証失敗
  - `unapproved_software_statement`: 未承認のソフトウェアステートメント

### 15-3. Client Configuration Endpoint（RFC 7592）

- [ ] `GET /{tenant_code}/register/{client_id}` の実装（クライアント情報取得）
  - Registration Access Token で認証
  - 現在のクライアントメタデータを返却
- [ ] `PUT /{tenant_code}/register/{client_id}` の実装（クライアント更新）
  - **仕様参照:** RFC 7592 Section 2.2
  - 全フィールドの送信が必要（差分更新ではない）
  - `client_id` は変更不可
  - Registration Access Token のローテーション（MAY → 実装する）
- [ ] `DELETE /{tenant_code}/register/{client_id}` の実装（クライアント削除）
  - HTTP 204 No Content
  - 関連するトークン・認可コードも失効させる

### 15-4. Discovery エンドポイントの更新

- [ ] `registration_endpoint` を discovery レスポンスに追加
  - **仕様参照:** OIDC Discovery 1.0 Section 3

### 15-5. 管理 API（Initial Access Token 管理）

- [ ] `POST /management/v1/tenants/:tenant_id/initial-access-tokens` — IAT 発行
- [ ] `GET /management/v1/tenants/:tenant_id/initial-access-tokens` — IAT 一覧
- [ ] `DELETE /management/v1/initial-access-tokens/:id` — IAT 無効化
- [ ] OP Frontend に IAT 管理画面を追加

### 15-6. RP での動作検証

- [ ] RP に Dynamic Registration デモページを追加
  - IAT を入力して自動登録を実行
  - 登録結果（client_id, client_secret）の表示
  - 登録したクライアントで認証フローを実行
  - クライアント情報の更新・削除

### 完了条件

- Initial Access Token を使ってクライアントを動的登録できる
- 登録したクライアントで Authorization Code Flow が動作する
- Registration Access Token でクライアント情報の取得・更新・削除ができる
- Software Statement の署名検証が動作する
- IAT なしの登録リクエストが拒否される
- `cd op/backend && go test ./...` が pass する

---

## Phase 16: Rich Authorization Requests — RAR（RFC 9396）

**目標:** scope の文字列ベースでは表現できない細粒度の認可要求を構造化 JSON で表現する仕組みを実装する。金融 API（FAPI）文脈での認可モデルを理解する。

**必読仕様:** RFC 9396 全文

### 学習ポイント

- **`scope` の限界**: `scope=transfer` では「どの口座から」「いくら」「誰に」を表現できない
- **`authorization_details` の構造**: `type` フィールドで認可の種別を識別し、任意のフィールドで詳細を記述
- **scope との共存**: `authorization_details` と `scope` は同一リクエストで併用可能
- **AS によるフィルタリング**: 要求された `authorization_details` を AS が精査し、実際に許可した内容をトークンレスポンスで返す

### 16-1. データモデル

- [ ] `authorization_detail_types` テーブルの作成
  - テナントごとにサポートする `type` を登録
  - `id`, `tenant_id`, `type_name`, `description`, `json_schema`（バリデーション用）
  - `allowed_actions`, `allowed_locations`（許可リスト）
- [ ] `authorization_codes` テーブルに `authorization_details` カラム追加
  - JSONB 型で認可要求の詳細を保存
- [ ] `access_tokens` テーブルに `authorization_details` カラム追加
  - 発行されたトークンに紐づく認可詳細

### 16-2. Authorization Details の処理

- [ ] `internal/oidc/rar.go` の作成
- [ ] `authorization_details` パラメータのパース
  - **仕様参照:** RFC 9396 Section 2（`authorization_details` Parameter）
  - URL エンコードされた JSON 配列をパース
  - 各オブジェクトの `type` フィールドの必須チェック
- [ ] 共通フィールドの処理（RFC 9396 Section 2 に定義）
  - `type`（REQUIRED）: 認可詳細のタイプ識別子
  - `locations`（OPTIONAL）: リソースサーバー URI の配列
  - `actions`（OPTIONAL）: 許可するアクションの配列
  - `datatypes`（OPTIONAL）: リクエストするデータタイプの配列
  - `identifier`（OPTIONAL）: 特定リソースの識別子
  - `privileges`（OPTIONAL）: 権限レベルの配列
- [ ] タイプごとのバリデーション
  - `authorization_detail_types` テーブルの `json_schema` で検証
  - サポートされていない `type` → エラー

### 16-3. 認可エンドポイントへの統合

- [ ] `authorize.go` に `authorization_details` 処理を追加
  - `scope` と `authorization_details` の両方を受け付ける
  - 認可コードに `authorization_details` を紐づけて保存
- [ ] 同意画面の拡張
  - `authorization_details` の内容を人間が読める形式で表示
  - ユーザーが個別の authorization detail を承認/拒否できる
- [ ] OP Frontend の同意画面を更新
  - scope に加えて authorization_details の詳細を表示

### 16-4. トークンエンドポイントへの統合

- [ ] トークンレスポンスに `authorization_details` を含める
  - **仕様参照:** RFC 9396 Section 7（Token Response）
  - AS が実際に許可した `authorization_details` を返却
  - リクエストされた内容と異なる場合がある（AS がフィルタリング）
- [ ] トークンリクエストでの `authorization_details` 制限
  - Refresh Token Grant 時に `authorization_details` でスコープを狭められる

### 16-5. Introspection / UserInfo への反映

- [ ] Token Introspection レスポンスに `authorization_details` を含める
- [ ] JWT アクセストークンのクレームに `authorization_details` を含める

### 16-6. Discovery エンドポイントの更新

- [ ] `authorization_details_types_supported` を追加
  - **仕様参照:** RFC 9396 Section 9（AS Metadata）

### 16-7. 管理 API・UI

- [ ] Authorization Detail Type の CRUD API
  - `GET /management/v1/tenants/:tenant_id/authorization-detail-types`
  - `POST /management/v1/tenants/:tenant_id/authorization-detail-types`
  - `PUT /management/v1/authorization-detail-types/:id`
  - `DELETE /management/v1/authorization-detail-types/:id`
- [ ] OP Frontend に設定画面を追加

### 16-8. デモシナリオ（RP）

金融 API を模した具体的なデモシナリオを実装し、RAR の価値を体感する。

- [ ] デモ用 authorization_detail type の定義
  - `payment_initiation`: 送金（金額・送金先・口座指定）
  - `account_information`: 口座情報照会（口座指定・期間指定）
- [ ] RP にデモページを追加
  - 「送金を許可する」シナリオ
    - `type: "payment_initiation"`, `actions: ["initiate"]`, `identifier: "account-123"` + 金額等のカスタムフィールド
  - 「口座情報を閲覧する」シナリオ
    - `type: "account_information"`, `actions: ["read"]`, `locations: ["https://example.com/accounts"]`
  - scope のみのリクエストと RAR リクエストの比較表示
  - トークンに含まれる `authorization_details` の可視化

### 完了条件

- `authorization_details` 付きの認可リクエストが処理できる
- 同意画面に authorization_details の詳細が表示される
- トークンレスポンスに許可された `authorization_details` が含まれる
- Introspection で `authorization_details` が返却される
- scope と authorization_details の併用が動作する
- RP のデモ UI で金融 API シナリオが体験できる
- `cd op/backend && go test ./...` が pass する

---

## 依存関係

```
Phase 13 (Conformance Test) ← 依存なし（既存実装のテスト）
Phase 14 (Token Exchange)   ← 依存なし
Phase 15 (Dynamic Client Registration) ← 依存なし
Phase 16 (RAR) ← 依存なし
```

各 Phase は独立しているため、興味のある順に着手可能。ただし Phase 13（Conformance Test）を先に実施すると、既存実装の仕様違反を発見・修正でき、後続の Phase の品質が上がる。

---

## 推奨実施順

1. **Phase 13**: 既存コードの品質保証が最優先。テストで仕様の穴を発見する
2. **Phase 14**: Token Exchange は token endpoint への追加なので、既存コードへの影響が最も小さい
3. **Phase 15**: Dynamic Client Registration は新しいエンドポイント群の追加
4. **Phase 16**: RAR は認可フロー・トークンフロー・同意画面・管理画面と影響範囲が広い

---

## マイグレーション追加予定

| Phase | テーブル |
|-------|----------|
| 14 | `token_exchange_policies` |
| 15 | `client_registrations`, `initial_access_tokens` |
| 16 | `authorization_detail_types`, `authorization_codes` に JSONB カラム追加, `access_tokens` に JSONB カラム追加 |
