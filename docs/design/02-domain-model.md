# ドメインモデル・DB設計

純粋なOIDC認証基盤として必要なエンティティとテーブル設計。
スコープ定義（[01-scope.md](./01-scope.md)）の「OPに含めるもの」に対応する。

---

## 1. ドメインモデル概要

```
┌─────────────────────────────────────────────────────────────┐
│                        OP（認証基盤）                        │
│                                                             │
│  ┌──────────┐   1:n   ┌──────────┐   1:n   ┌───────────┐  │
│  │  Tenant  │────────▶│  Client  │────────▶│ RedirectUri│  │
│  │ (テナント) │         │  (RP登録) │         │           │  │
│  └──────────┘         └────┬─────┘         └───────────┘  │
│        │ 1:n               │                               │
│        ▼                   │                               │
│  ┌──────────┐              │ n:n                           │
│  │   User   │◀─────────────┘                               │
│  │ (ユーザー) │                                              │
│  └─────┬────┘                                              │
│        │                                                   │
│        ├──── 1:n ──▶ [ Credential ]  パスワード/外部IdP     │
│        ├──── 1:1 ──▶ [ MfaConfig ]   TOTP/パスキー設定      │
│        ├──── 1:n ──▶ [ Session ]     ログインセッション      │
│        └──── 1:n ──▶ [ Token ]       発行済みトークン        │
│                                                            │
│  ┌──────────┐                                              │
│  │ SignKey  │  JWT署名鍵（テナント共有 or テナント別）         │
│  └──────────┘                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. エンティティ定義

### 2-1. Tenant（テナント）

マルチテナント構成の単位。金融機関・企業・組織など「このOPを利用する組織」に対応する。

**含めるもの:**
- テナントを識別するコード・名称
- トークン有効期限設定（テナントごとに異なるセキュリティポリシーを許容）
- セッション有効期限設定

**含めないもの:**
- テナント内の組織構造（企業・部門）→ RPが管理
- テナント固有のビジネス設定 → RPが管理

```
tenants
├── id: uuid (PK)
├── code: string (unique)        ← URLに含めるテナント識別子
├── name: string
├── session_lifetime: int        ← 認証セッション有効期限（秒）
├── auth_code_lifetime: int      ← 認可コード有効期限（秒）
├── access_token_lifetime: int
├── refresh_token_lifetime: int
├── id_token_lifetime: int
├── created_at
└── updated_at
```

### 2-2. Client（OIDCクライアント / RP）

このOPに登録されたRP。OIDC仕様の「Relying Party」に対応する。

```
clients
├── id: uuid (PK)
├── tenant_id: uuid (FK → tenants)
├── client_id: string (unique)   ← OIDC仕様のclient_id
├── client_secret_hash: string   ← ハッシュ化して保存
├── name: string
├── grant_types: json            ← 許可するフロー ["authorization_code", ...]
├── response_types: json         ← ["code", "token", ...]
├── token_endpoint_auth_method: enum  ← client_secret_post / client_secret_basic / none
├── require_pkce: boolean
├── frontchannel_logout_uri: string|null   ← Front-Channel Logout用
├── backchannel_logout_uri: string|null    ← Back-Channel Logout用
├── status: enum                 ← active / disabled
├── created_at
└── updated_at

redirect_uris
├── id: uuid (PK)
├── client_id: uuid (FK → clients)
├── uri: string
└── created_at

post_logout_redirect_uris
├── id: uuid (PK)
├── client_id: uuid (FK → clients)
├── uri: string
└── created_at
```

### 2-3. User（ユーザー）

OPが管理するユーザー。**OIDC標準クレームに対応する属性と、認証に必要な属性のみ持つ。**

```
users
├── id: uuid (PK)                ← OIDC仕様の「sub」クレームに使用
├── tenant_id: uuid (FK → tenants)
├── login_id: string (unique per tenant)
├── email: string
├── email_verified: boolean
├── name: string|null            ← OIDC標準クレーム（profile スコープ）
├── status: enum                 ← active / locked / disabled
├── last_login_at: timestamp|null
├── created_at
└── updated_at
```

> **`name`を持つかどうかは設計判断が必要。**
>
> OIDC Core 1.0 Section 5.4 で `profile` スコープのクレームとして `name` が定義されており、
> RPから `profile` スコープを要求された際にuserinfoで返せることがOPに期待される。
>
> - **OPに`name`を持たせる**: OIDC標準クレームとして妥当。ただし「認証に必要か」という観点ではRP側の属性と言える
> - **OPに`name`を持たせない**: RPが`sub`をキーに自身のDBで管理する。`profile`スコープには応答しない設計になる
>
> **どちらを選んでも正当だが、設計初期に方針を決めて明記すること。**
> 本ドキュメントでは「OIDC標準クレーム（name, email等）はOPが管理する」方針を採用する。
>
> **含めないもの（どちらの方針でも共通）**: `department_id`・`role`・`ruby`等のRP固有業務属性。
> これらはRP側が`sub`をキーとして自身のDBで管理する。

### 2-4. Credential（認証情報）

ユーザーの認証手段。複数の認証方式に対応するためポリモーフィックな設計にする。

```
credentials
├── id: uuid (PK)
├── user_id: uuid (FK → users)
├── type: enum              ← password / oidc_provider（外部IdP連携）
├── created_at
└── updated_at

password_credentials
├── id: uuid (PK)
├── credential_id: uuid (FK → credentials)
├── password_hash: string
├── algorithm: string       ← bcrypt / argon2id 等
└── updated_at

password_histories           ← パスワード再利用防止
├── id: uuid (PK)
├── user_id: uuid (FK → users)
├── password_hash: string
└── created_at

external_idp_credentials     ← 外部IdP（Google等）との連携
├── id: uuid (PK)
├── credential_id: uuid (FK → credentials)
├── provider: string        ← google / line 等
├── provider_subject: string ← 外部IdPのsub
└── created_at
```

### 2-5. MfaConfig（MFA設定）

ユーザーのMFA設定。認証方式ごとに分離する。

```
mfa_configs
├── id: uuid (PK)
├── user_id: uuid (FK → users)
├── type: enum              ← totp / webauthn
├── enabled: boolean
├── verified_at: timestamp|null
├── created_at
└── updated_at

totp_configs
├── id: uuid (PK)
├── mfa_config_id: uuid (FK → mfa_configs)
├── secret_key_encrypted: string  ← 暗号化して保存（KMS等）
├── algorithm: string       ← SHA1 / SHA256
├── digits: int             ← 6
├── period: int             ← 30
└── last_used_step: int|null  ← リプレイ攻撃防止

webauthn_credentials        ← パスキー対応
├── id: uuid (PK)
├── mfa_config_id: uuid (FK → mfa_configs)
├── credential_id: string   ← WebAuthnのcredential ID
├── public_key: text
├── sign_count: int
└── created_at
```

### 2-6. Session（セッション）

ユーザーのログインセッション。SSOセッションの管理に使う。

```
sessions
├── id: uuid (PK)            ← OIDC仕様の「sid」クレームに使用
├── user_id: uuid (FK → users)
├── tenant_id: uuid (FK → tenants)
├── ip_address: string
├── user_agent: string
├── expires_at: timestamp
├── revoked_at: timestamp|null
├── created_at
└── updated_at
```

> `sid`は Back-Channel Logout の logout_token にも含まれる。セッション単位でのログアウト通知に必要。

### 2-7. トークン群

発行されたトークンを管理する。検証・失効の両方に使う。

```
authorization_codes          ← 認可コード（使い捨て）
├── id: uuid (PK)
├── session_id: uuid (FK → sessions)
├── client_id: uuid (FK → clients)
├── code: string (unique)
├── redirect_uri: string     ← 発行時のredirect_uri（検証用）
├── scope: string
├── nonce: string|null
├── code_challenge: string|null
├── code_challenge_method: string|null
├── expires_at: timestamp    ← RFC 6749より「最大10分」がRECOMMENDED。実用上は1〜2分が多い
└── used_at: timestamp|null  ← 使用済みフラグ（nullなら未使用）
                                使用済みコードの再提示はMUST拒否 + 既発行トークンをSHOULD失効

access_tokens
├── id: uuid (PK)
├── jti: string (unique)     ← JWT IDクレーム
├── session_id: uuid (FK → sessions)
├── client_id: uuid (FK → clients)
├── scope: string
├── expires_at: timestamp
└── revoked_at: timestamp|null

refresh_tokens
├── id: uuid (PK)
├── token_hash: string (unique)    ← ハッシュ化して保存
├── parent_id: uuid|null (FK → refresh_tokens)  ← Reuse Detection用（前のトークンへの参照）
├── session_id: uuid (FK → sessions)            ← 再利用検知時にセッション全体を失効させるために必要
├── access_token_id: uuid (FK → access_tokens)
├── expires_at: timestamp
├── revoked_at: timestamp|null
└── reuse_detected_at: timestamp|null  ← 再利用検知時刻（監査用）

id_tokens                    ← 発行履歴（検証・監査用）
├── id: uuid (PK)
├── jti: string (unique)
├── session_id: uuid (FK → sessions)
├── client_id: uuid (FK → clients)
├── nonce: string|null
├── expires_at: timestamp
└── created_at
```

### 2-8. SignKey（署名鍵）

JWT署名用の鍵ペア。鍵ローテーションを考慮する。

```
sign_keys
├── id: uuid (PK)
├── kid: string (unique)         ← JWT headerの"kid"クレーム
├── algorithm: string            ← RS256 / ES256
├── public_key: text             ← PEM形式
├── private_key_ref: string      ← 秘密鍵への参照（KMS keyID等）
│                                   秘密鍵自体はDBに保存しない
├── active: boolean
├── created_at
└── rotated_at: timestamp|null
```

> **秘密鍵はDBに保存しない。** AWS KMS等の鍵管理サービスへの参照のみ保持する。

---

## 3. テーブル関係の全体像

```
tenants
  │ 1:n
  ├──▶ clients
  │      │ 1:n
  │      ├──▶ redirect_uris
  │      └──▶ post_logout_redirect_uris
  │
  └──▶ users
         │ 1:n
         ├──▶ credentials
         │      ├──▶ password_credentials
         │      │      └── password_histories (via user_id)
         │      └──▶ external_idp_credentials
         │
         ├──▶ mfa_configs
         │      ├──▶ totp_configs
         │      └──▶ webauthn_credentials
         │
         └──▶ sessions
                │ 1:n
                ├──▶ authorization_codes
                ├──▶ access_tokens
                │      └──▶ refresh_tokens
                └──▶ id_tokens

sign_keys  （テナントとは独立。OPレベルで管理）
```

---

## 4. 設計上の重要な判断

### 判断1: Sessionエンティティを明示的に持つ

多くのシンプルなOIDC実装ではセッションをフレームワークのセッション機能に任せるが、以下の理由でDBに明示的に持つことを推奨する：

- SLO（シングルログアウト）で`sid`クレームを使った通知が必要
- 「他端末でのセッション一覧・強制ログアウト」機能に対応できる
- 監査ログとしてのログイン履歴を残せる

### 判断2: refresh_tokenはハッシュ化して保存

アクセストークン（JWT）と異なり、リフレッシュトークンはランダム文字列。そのまま保存するとDB流出時に全て悪用される。ハッシュ化して保存し、検証時にハッシュを比較する。

### 判断3: 認可コードの`used_at`で二重利用を防ぐ

`used_at`がnullでないコードは拒否する。削除ではなくupdateで使用済みフラグを立てることで、監査ログとして履歴を残す。

### 判断4: MFAはテナント設定ではなくユーザー設定として持つ

「テナント単位でMFA必須」の設定は別途テナントポリシーとして持ち（`tenants`テーブルの`mfa_required`フラグ等）、ユーザーの実際のMFA設定は`mfa_configs`に分離する。

### 判断5: usersテーブルにRP業務属性を持たせない

`users`テーブルは認証に必要な最小構成に留める。KKPPの反省として、`department_id`や`ruby`等の業務属性を同じテーブルに混在させると、OP/RPの責務分離が崩れる起点になる。

---

## 5. インデックス設計方針

認証フローでの検索パターンに合わせてインデックスを設計する。

| テーブル | インデックス対象 | 理由 |
|---------|----------------|------|
| `users` | `(tenant_id, login_id)` | ログイン時の検索 |
| `users` | `email` | パスワードリセット時の検索 |
| `clients` | `client_id` | 認可リクエスト受付時の検索 |
| `authorization_codes` | `code` | トークンエンドポイントでの検索 |
| `access_tokens` | `jti` | トークン検証時の検索 |
| `refresh_tokens` | `token_hash` | リフレッシュ時の検索 |
| `sessions` | `(user_id, revoked_at)` | SLO・セッション一覧の検索 |
| `sign_keys` | `kid` | JWT検証時の公開鍵取得 |
