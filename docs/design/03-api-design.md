# APIインターフェース設計

純粋なOIDC認証基盤として外部に公開するAPIの設計指針。
「OIDC標準エンドポイント」と「運用管理API」を明確に分ける。

---

## 1. APIの分類

| 分類 | 対象 | 認証方式 |
|------|------|---------|
| **OIDCエンドポイント** | RP（クライアント）・エンドユーザー | OIDC仕様に準拠 |
| **OP管理API** | OP運用者（Kokopelli等） | 管理者トークン / APIキー |
| **OP内部API** | フロントエンド（ログイン画面） | セッションCookie |

RPに公開する業務管理APIは作らない。（スコープ定義の原則に従う）

---

## 2. OIDCエンドポイント

### 2-1. Discoveryエンドポイント

```
GET /.well-known/openid-configuration
GET /{tenant_code}/.well-known/openid-configuration  ← マルチテナントの場合
```

RPが自動設定のために最初に取得するメタ情報。以下を返す。

```json
{
  "issuer": "https://idp.example.com/{tenant_code}",
  "authorization_endpoint": "https://idp.example.com/{tenant_code}/authorize",
  "token_endpoint": "https://idp.example.com/{tenant_code}/token",
  "userinfo_endpoint": "https://idp.example.com/{tenant_code}/userinfo",
  "jwks_uri": "https://idp.example.com/jwks",
  "end_session_endpoint": "https://idp.example.com/{tenant_code}/logout",
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code", "refresh_token", "client_credentials"],
  "subject_types_supported": ["public"],
  "id_token_signing_alg_values_supported": ["RS256"],
  "scopes_supported": ["openid", "profile", "email", "offline_access"],
  "token_endpoint_auth_methods_supported": ["client_secret_basic", "client_secret_post"],
  "code_challenge_methods_supported": ["S256"],
  "backchannel_logout_supported": true,
  "frontchannel_logout_supported": true
}
```

**`issuer`のURL設計について（重要）:**

OIDC Discovery 1.0 Section 4.3 より、Discoveryで返す`issuer`の値はDiscoveryリクエストに使ったURLのプレフィックスと完全一致しなければならない（MUST）。またIDトークンの`iss`クレームとも一致する。

マルチテナント構成での`issuer`設計は2つの方式があり、どちらも仕様上有効だが**一度決めたら変更不可**。

| 方式 | issuer例 | Discovery URL例 | 特徴 |
|------|---------|----------------|------|
| **パス方式** | `https://idp.example.com/tenant1` | `https://idp.example.com/tenant1/.well-known/openid-configuration` | 単一ドメインで運用可能 |
| **サブドメイン方式** | `https://tenant1.idp.example.com` | `https://tenant1.idp.example.com/.well-known/openid-configuration` | テナントごとにDNS・証明書が必要 |

仕様上の注意（OIDC Discovery 1.0 Section 4.1）:
> If the Issuer value contains a path component, any terminating `/` MUST be removed before appending `/.well-known/openid-configuration`.

つまりパス方式の場合、末尾スラッシュの扱いを統一しないと検証エラーになる。`issuer`には末尾スラッシュを含めない。

### 2-2. 認可エンドポイント

```
GET /{tenant_code}/authorize
```

**パラメータ（OIDC仕様準拠）:**

| パラメータ | 必須 | 説明 |
|-----------|------|------|
| `response_type` | 必須 | `code`（Authorization Code Flow推奨） |
| `client_id` | 必須 | 登録済みクライアントID |
| `redirect_uri` | 必須 | 登録済みURIと完全一致 |
| `scope` | 必須 | `openid`を含むスペース区切りのスコープ |
| `state` | 推奨 | CSRF対策（RPが生成するランダム値） |
| `nonce` | 条件付き必須 | Authorization Code FlowではOPTIONAL。Implicit Flowでは**REQUIRED**。Hybrid Flowは`response_type`による（`code id_token`等はREQUIRED）。送った場合はIDトークン内で一致検証が必須（MUST） |
| `code_challenge` | 推奨 | PKCE（S256メソッド） |
| `code_challenge_method` | 推奨 | `S256` |
| `prompt` | 任意 | `none` / `login` / `consent` |
| `max_age` | 任意 | 最大認証経過時間（秒） |

**処理フロー:**

```
1. client_id, redirect_uri の登録確認
2. セッション確認（SSO）
   - 有効なセッションあり → 認可コード発行 → redirect
   - セッションなし → ログイン画面へ（内部リダイレクト）
3. ログイン完了後 → 認可コード発行 → redirect_uri へリダイレクト
```

**成功レスポンス（redirect）:**
```
{redirect_uri}?code={authorization_code}&state={state}
```

**エラーレスポンス（redirect）:**
```
{redirect_uri}?error={error_code}&error_description={description}&state={state}
```

### 2-3. トークンエンドポイント

```
POST /{tenant_code}/token
Content-Type: application/x-www-form-urlencoded
Authorization: Basic {client_id:client_secret の Base64}  ← client_secret_basic の場合
```

**grant_type別のパラメータ:**

#### authorization_code
```
grant_type=authorization_code
&code={authorization_code}
&redirect_uri={redirect_uri}          ← 認可リクエスト時と同一
&client_id={client_id}                ← client_secret_post の場合
&client_secret={client_secret}        ← client_secret_post の場合
&code_verifier={code_verifier}        ← PKCE使用時
```

#### refresh_token
```
grant_type=refresh_token
&refresh_token={refresh_token}
&scope={scope}                        ← 任意（元スコープ以下に限定）
```

> **`offline_access`スコープとRefresh Tokenの関係:**
> OIDC Core 1.0 Section 11 より、`offline_access`スコープはユーザーが**不在時**（ログアウト後）もアクセストークンを更新し続けるユースケースのためのもの。このスコープを要求する場合は`prompt=consent`が必要（MUST）。
>
> ただし同仕様の末尾には "The Authorization Server **MAY** grant Refresh Tokens in other contexts that are beyond the scope of this specification." と記載されており、**OPの裁量で`offline_access`なしでもRefresh Tokenを発行できる**（セッション中のみ有効なRefresh Token等）。
>
> **設計判断として明記すべき事項:** 自プロダクトでどちらの方針を取るか決め、クライアント登録設定に反映する。

#### client_credentials
```
grant_type=client_credentials
&scope={scope}
```

**成功レスポンス:**
```json
{
  "access_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "...",             ← offline_access スコープ時のみ
  "id_token": "eyJ...",              ← openid スコープ時のみ
  "scope": "openid profile"
}
```

**検証チェックリスト（実装時に必ず確認）:**

- [ ] `code`の有効期限確認
- [ ] `code`の使用済み確認（二重使用防止）
- [ ] `redirect_uri`が認可リクエスト時と一致
- [ ] PKCE: `code_verifier`から`code_challenge`を再計算して一致確認
- [ ] クライアント認証の確認

### 2-4. userinfoエンドポイント

```
GET /{tenant_code}/userinfo
Authorization: Bearer {access_token}
```

**レスポンス（要求スコープに応じたクレームを返す）:**
```json
{
  "sub": "user-uuid",
  "email": "user@example.com",         ← email スコープ
  "email_verified": true,              ← email スコープ
  "name": "山田太郎",                   ← profile スコープ
  "updated_at": 1700000000             ← profile スコープ
}
```

> `sub`以外のクレームは要求されたスコープに応じてのみ返す。OPが管理しない属性（部門・役職等）はここに含まない。

### 2-5. トークン失効エンドポイント

```
POST /{tenant_code}/revoke
Content-Type: application/x-www-form-urlencoded
```

```
token={token}
&token_type_hint={access_token|refresh_token}  ← 任意
```

- アクセストークンの失効: そのトークンのみ
- リフレッシュトークンの失効: そのリフレッシュトークンと紐付くアクセストークンも失効
- 存在しないトークンを指定しても`200 OK`を返す（仕様上、エラーにしない）

### 2-4-a. Refresh Token Rotation + Reuse Detection

RFC 9700 Section 4.14.2 より、公開クライアントに対してはRefresh Token Rotationの実装が必須（MUST）。機密クライアントに対しても強く推奨される。

**ローテーションの動作:**

```
1. クライアントがリフレッシュトークン（RT-1）でアクセストークンを要求
2. OPは新しいリフレッシュトークン（RT-2）を発行し、RT-1を無効化
3. クライアントはRT-2を保持する
```

**Reuse Detection（再利用検知）の動作:**

単なるローテーション（古いトークンを無効化するだけ）では不十分。無効化済みのトークンが再利用された場合、トークン漏洩の可能性があるため、**紐付くセッション全体を失効**させる。

```
正常ケース:
  クライアント → RT-1 提示 → RT-2発行、RT-1無効化

漏洩検知ケース:
  攻撃者   → RT-1 提示 → RT-2発行、RT-1無効化
  正規客 → RT-1 提示（既に無効）→ 【RT-2も含む関連セッション全体を失効】
                                  → 正規クライアントに再認証を要求
```

RFC 9700 原文:
> If a refresh token is compromised and subsequently used by both the attacker and the legitimate client, one of them will present an invalidated refresh token, which will inform the authorization server of the breach. The authorization server cannot determine which party submitted the invalid refresh token, but it will revoke the active refresh token.

**DB設計との対応:**

`refresh_tokens`テーブルに`parent_token_id`（前のトークンへの参照）を持たせることで、再利用検知時に同一チェーン上のトークンを一括失効できる。

```
refresh_tokens
├── id
├── parent_id: uuid|null   ← 前のリフレッシュトークンのID
├── session_id             ← セッション全体の失効に使う
├── revoked_at
└── reuse_detected_at      ← 再利用検知時刻（監査用）
```

### 2-6. JWKSエンドポイント

```
GET /jwks
```

RPがJWT署名を検証するための公開鍵群を返す。鍵ローテーション中は複数の鍵を返す。

```json
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "2024-01-key",
      "use": "sig",
      "alg": "RS256",
      "n": "...",
      "e": "AQAB"
    }
  ]
}
```

### 2-7. ログアウトエンドポイント（RP-Initiated Logout）

```
GET /{tenant_code}/logout
POST /{tenant_code}/logout
```

**パラメータ:**

| パラメータ | 必須 | 説明 |
|-----------|------|------|
| `id_token_hint` | 推奨 | ログアウト対象セッション特定のためのIDトークン |
| `post_logout_redirect_uri` | 任意 | ログアウト完了後のリダイレクト先（登録済みURIのみ） |
| `state` | 任意 | CSRF対策（post_logout_redirect_uri 使用時に推奨） |
| `logout_hint` | 任意 | ログアウト対象ユーザーのヒント（OIDC Session Management拡張） |

**処理フロー:**

```
1. id_token_hint 検証（iss, aud, 署名）
2. セッション特定・無効化
3. 関連トークン失効（access_token, refresh_token）
4. Front-Channel Logout 実行（登録済みRPに通知）
5. Back-Channel Logout 実行（登録済みRPにサーバー間通知）
6. post_logout_redirect_uri へリダイレクト（指定時）
```

**冪等性:** 未認証・セッション不明の状態でもエラーにしない。

---

## 3. SLO関連エンドポイント

### 3-1. Front-Channel Logoutの仕組み

OP側のフロントエンドが各RPの`frontchannel_logout_uri`にiframeでGETリクエストを送る。

```
GET {frontchannel_logout_uri}?iss={issuer}&sid={session_id}
```

- `iss`: OPのissuer識別子
- `sid`: セッションID（RPがセッション特定に使う）

**`iss`・`sid`の送信要件（OIDC Front-Channel Logout 1.0より）:**

> "An iss query parameter and a sid query parameter **MAY** be included by the OP. If either is included, both **MUST** be."

つまり送信は任意（MAY）だが、**片方だけ送ることは禁止（両方セットで送るか、送らないか）**。RP側での検証も任意（MAY）。

**OP側の実装方針:**
- ログアウト完了ページでJavaScriptによりiframeを生成
- 各RPのURLへリクエスト送信
- ブラウザが閉じている場合は通知できない（Back-Channel Logoutで補完）

### 3-2. Back-Channel Logoutの仕組み

OPがサーバー間でRPの`backchannel_logout_uri`にPOSTする。

```
POST {backchannel_logout_uri}
Content-Type: application/x-www-form-urlencoded

logout_token={signed_jwt}
```

**logout_tokenのクレーム:**
```json
{
  "iss": "https://idp.example.com/{tenant_code}",
  "sub": "user-uuid",          ← subとsidはどちらか一方が必須（MUST）、両方可（MAY）
  "aud": "client-id",
  "iat": 1700000000,
  "exp": 1700000300,           ← 必須（MUST）
  "jti": "unique-token-id",
  "events": { "http://schemas.openid.net/event/backchannel-logout": {} },
  "sid": "session-uuid"        ← subとsidはどちらか一方が必須（MUST）、両方可（MAY）
}
```

**logout_tokenの仕様要件（OIDC Back-Channel Logout 1.0より）:**

| クレーム | 要件 |
|---------|------|
| `iss` | REQUIRED |
| `aud` | REQUIRED |
| `iat` | REQUIRED |
| `exp` | REQUIRED |
| `jti` | REQUIRED（リプレイ攻撃防止） |
| `events` | REQUIRED |
| `sub` | `sub`または`sid`のどちらか一方は**MUST**。両方含めてもよい（MAY） |
| `sid` | `sub`または`sid`のどちらか一方は**MUST**。両方含めてもよい（MAY） |
| `nonce` | **MUST NOT**（構文的に認証レスポンスへの偽造利用を防ぐため禁止） |

- `sid`のみの場合: そのセッションのみログアウト
- `sub`のみの場合: そのユーザーのRP上の全セッションをログアウト
- 両方ある場合: 特定セッションのログアウト（最も精密）

**推奨:** `typ: "logout+jwt"` ヘッダーを付与することで、logout_tokenが認証レスポンスとして誤用されることをさらに防止できる。

---

## 4. OP管理API

OPを運用するための管理APIはOIDCエンドポイントとパスを明確に分ける。

```
/management/v1/*   ← 管理API専用プレフィックス
```

### 4-1. クライアント管理

```
GET    /management/v1/tenants/{tenant_id}/clients       ← クライアント一覧
POST   /management/v1/tenants/{tenant_id}/clients       ← クライアント登録
GET    /management/v1/clients/{client_id}               ← クライアント詳細
PUT    /management/v1/clients/{client_id}               ← クライアント更新
DELETE /management/v1/clients/{client_id}               ← クライアント無効化

PUT    /management/v1/clients/{client_id}/secret        ← シークレットローテーション
GET    /management/v1/clients/{client_id}/redirect-uris
POST   /management/v1/clients/{client_id}/redirect-uris
DELETE /management/v1/clients/{client_id}/redirect-uris/{id}
```

### 4-2. テナント管理

```
GET    /management/v1/tenants
POST   /management/v1/tenants
GET    /management/v1/tenants/{tenant_id}
PUT    /management/v1/tenants/{tenant_id}
```

### 4-3. 鍵管理

```
GET    /management/v1/keys                 ← 鍵一覧
POST   /management/v1/keys/rotate         ← 鍵ローテーション
DELETE /management/v1/keys/{kid}          ← 鍵の無効化
```

### 4-4. インシデント対応

```
POST   /management/v1/incidents/revoke-all-tokens        ← 全トークン失効
POST   /management/v1/incidents/revoke-tenant-tokens     ← テナント全トークン失効
POST   /management/v1/incidents/revoke-user-tokens       ← ユーザー全トークン失効
```

---

## 5. OP内部API（フロントエンド向け）

ログイン画面（フロントエンド）がバックエンドを呼ぶ内部API。**外部には公開しない。**

```
POST   /internal/login                    ← ログイン（ID/パスワード）
POST   /internal/logout                   ← ログアウト
GET    /internal/me                       ← 現在のセッション確認
POST   /internal/password/reset-request   ← パスワードリセットメール送信
POST   /internal/password/reset           ← パスワードリセット実行
POST   /internal/mfa/totp/verify          ← TOTP検証
POST   /internal/mfa/totp/setup           ← TOTP初期設定
```

**設計上の原則:**

- `/internal/*`のエンドポイントはCORSで外部オリジンからのアクセスを禁止
- RPに公開するAPIはOIDCエンドポイントのみ
- ユーザーのCRUD（作成・更新・削除）はこの内部APIにも含めない

---

## 6. カスタム拡張の判断基準

OIDC仕様外のカスタムエンドポイントを追加する際の判断基準。

### 追加してよいカスタム拡張

| 拡張 | 理由 |
|------|------|
| フロントエンド分離のための内部API | 実装上の都合（仕様には影響しない） |
| Discovery以外のメタ情報取得 | OP固有の設定情報提供 |
| セッション一覧取得（ユーザー向け） | セキュリティ機能として合理的 |

### 追加してはいけないカスタム拡張

| 拡張 | 理由 |
|------|------|
| RP業務データの取得API | RPの責務をOPに持ち込む |
| ユーザーのCRUD API（外部公開） | ユーザー管理はRPが行う |
| テナント固有のビジネスロジックAPI | OPの汎用性が失われる |

### カスタムクレームの追加

RPからカスタムクレームの要求が来た場合の対応方針:

1. **まず断る**: RPが`sub`を元に自身のDBから取得する設計を提案する
2. **スコープ拡張で対応**: どうしても必要なら、クライアント登録時にカスタムスコープを定義し、そのスコープ要求時のみクレームを付与する
3. **OP側のDBには最小限のみ追加**: 複数のRPで共通して必要なクレームに限定する

---

## 7. エラーレスポンス設計

OIDC仕様のエラーコードに準拠する。

### OIDCエンドポイントのエラー

```json
{
  "error": "invalid_request",
  "error_description": "The request is missing a required parameter."
}
```

**主なエラーコード:**

| エラーコード | 説明 |
|------------|------|
| `invalid_request` | 必須パラメータの欠如・形式エラー |
| `unauthorized_client` | クライアントが該当grant_typeを使用不可 |
| `access_denied` | ユーザーが認可を拒否 |
| `unsupported_response_type` | 未対応のresponse_type |
| `invalid_scope` | 無効なスコープ |
| `server_error` | OPの内部エラー |
| `invalid_client` | クライアント認証失敗 |
| `invalid_grant` | 無効な認可コード・リフレッシュトークン |

### 管理APIのエラー

管理APIはREST標準のHTTPステータスコードを使う。

| ステータス | 用途 |
|-----------|------|
| `400 Bad Request` | リクエスト形式エラー |
| `401 Unauthorized` | 管理者認証失敗 |
| `403 Forbidden` | 権限なし |
| `404 Not Found` | リソースが存在しない |
| `409 Conflict` | 重複リソース |
| `500 Internal Server Error` | サーバーエラー |

---

## 8. セキュリティ設計チェックリスト

APIを実装する際に確認すべきセキュリティ項目。

### OIDCエンドポイント共通

- [ ] `redirect_uri`は登録済みURIと完全一致（前方一致不可）
- [ ] `state`パラメータの検証をRP側に要求（ドキュメントに明記）
- [ ] HTTPSのみで提供（HTTPは拒否）
- [ ] `iss`クレームのURL末尾スラッシュを統一する（`issuer`の厳密な一致）

### トークンエンドポイント

- [ ] クライアント認証をすべてのリクエストで実施
- [ ] 認可コードの一回限り使用を強制（`used_at`チェック）
- [ ] PKCE検証（S256のみサポート推奨。RFC 7636より「S256が使えるクライアントはMUST use S256」「サーバー側のS256はMTI（必須実装）」。`plain`は後方互換のためのみ存在し、新規実装では原則サポート不要）
- [ ] リフレッシュトークンのローテーション（使用のたびに新しいトークンを発行）+ Reuse Detection
- [ ] レート制限（ブルートフォース対策）

### ログアウトエンドポイント

- [ ] `id_token_hint`の署名・`iss`・`aud`を検証
- [ ] `post_logout_redirect_uri`は登録済みURIと完全一致
- [ ] Back-Channel Logout通知の失敗をログに記録（通知失敗でもユーザーをブロックしない）
- [ ] RP-Initiated Logoutの発起元RPも含めてFront/Back-Channel通知する（仕様上の要件）
