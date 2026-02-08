# Phase 1: OPコア（認証基盤の骨格）実装計画

## Context

Phase 0（開発環境構築）が完了し、Docker Compose で全サービスが起動する状態。Phase 1 では Authorization Code Flow（PKCE付き）を一通り動作させ、RPがOPを経由してログインできる最小構成を構築する。

9つのサブフェーズに分けて段階的に実装・検証を行う。

---

## 依存パッケージ追加（1-1 で実施）

```
gorm.io/gorm + gorm.io/driver/postgres
github.com/golang-migrate/migrate/v4
github.com/lestrrat-go/jwx/v3
github.com/google/uuid
golang.org/x/crypto（直接依存に昇格 - argon2id用）
```

## 環境変数追加

- `OP_KEY_ENCRYPTION_KEY` - 署名鍵の暗号化キー（64 hex chars = 32 bytes）
- `.env.example`, `docker-compose.yml` に追加

---

## サブフェーズ 1-1: DB・基盤

**目標:** マイグレーション基盤、全テーブル作成、GORM初期化、ドメインモデル、リポジトリ

### 作成ファイル
| ファイル | 内容 |
|---------|------|
| `config/config.go` | Config struct + Load() - 環境変数の一元管理 |
| `migrations/000001_create_tables.up.sql` | 14テーブル一括作成（02-domain-model.md準拠） |
| `migrations/000001_create_tables.down.sql` | 全テーブルDROP |
| `migrations/000002_seed_data.up.sql` | demo テナント + testuser |
| `internal/database/database.go` | GORM初期化 |
| `internal/database/migrate.go` | golang-migrate実行 |
| `internal/model/*.go` | 7ファイル: tenant, client, user, credential, session, token, sign_key |
| `internal/store/*.go` | 9ファイル: GORM実装 |
| `cmd/server/main.go` | 変更: Config, DB接続, マイグレーション実行 |

### テーブル一覧（op スキーマ）
tenants, clients, redirect_uris, post_logout_redirect_uris, users, credentials, password_credentials, password_histories, sessions, authorization_codes, access_tokens, refresh_tokens, id_tokens, sign_keys

### 検証
- `docker compose --profile op up --build` → 全テーブル作成確認
- シードデータ（demo テナント、testuser）存在確認
- `/healthz` 継続動作

---

## サブフェーズ 1-2: JWT署名鍵管理

**目標:** RSA鍵ペア生成、AES-GCM暗号化保存、JWKSエンドポイント

### 設計方針
- 秘密鍵は AES-256-GCM で暗号化して `sign_keys.private_key_ref` に保存
- 暗号化キーは環境変数 `OP_KEY_ENCRYPTION_KEY`
- RSA 2048ビット、kid形式: `{date}-{random8hex}`
- `jwt/deps.go` の KeyService interface で将来KMS差替え可能

### 作成ファイル
| ファイル | 内容 |
|---------|------|
| `internal/crypto/aes.go` | AES-256-GCM 暗号化/復号 |
| `internal/jwt/key_service.go` | KeyService実装（鍵生成・保存・取得・JWKSet） |
| `internal/jwt/deps.go` | KeyService interface |
| `internal/oidc/jwks.go` | `GET /jwks` |
| `migrations/000003_seed_client.up.sql` | demo-rp クライアント + redirect_uri |

### 検証
- `curl http://localhost:8080/jwks` → JWK Set（kty, kid, use, alg, n, e）
- 秘密鍵パラメータ（d, p, q等）がレスポンスに含まれないこと
- DB内の `private_key_ref` が暗号化されていること
- サービス再起動で同じ鍵が使われること

---

## サブフェーズ 1-3: Discoveryエンドポイント

**目標:** OIDC Discovery 1.0 準拠メタデータ

### 作成ファイル
| ファイル | 内容 |
|---------|------|
| `internal/oidc/discovery.go` | `GET /{tenant_code}/.well-known/openid-configuration` |

### 仕様準拠
- `issuer` 末尾スラッシュなし（MUST: Section 4.1）
- `issuer` は Discovery URL プレフィックスと完全一致（MUST: Section 4.3）

### 検証
- `curl http://localhost:8080/demo/.well-known/openid-configuration`
- 存在しないテナント → 404

---

## サブフェーズ 1-4: 認証フロー（内部API）

**目標:** argon2idパスワードハッシュ、ログイン、セッション管理、ログインUI

### セッション管理
- セッションIDはDB `sessions.id`（UUID）
- `op_session` Cookie（HttpOnly, SameSite=Lax）
- 開発環境では Secure フラグを外す

### 作成ファイル
| ファイル | 内容 |
|---------|------|
| `internal/crypto/argon2id.go` | argon2id ハッシュ・検証 |
| `internal/auth/login.go` | Login ハンドラ + ロジック |
| `internal/auth/me.go` | Me ハンドラ + ロジック |
| `internal/auth/deps.go` | AuthService interface |
| `internal/auth/errors.go` | 認証エラー定義 |
| `cmd/seed/main.go` | パスワードハッシュ生成 + シードデータ投入 |
| `migrations/000004_seed_credentials.up.sql` | testuser のパスワード credential |
| `op/frontend/src/app/login/page.tsx` | 最小限ログインフォーム |

### CORS設定
OP Frontend (localhost:3000) → OP Backend (localhost:8080) の通信許可

### 検証
- `POST /internal/login` → 200 + Set-Cookie: op_session=...
- `GET /internal/me` with Cookie → セッション情報
- 不正パスワード → 401
- ログイン画面 `http://localhost:3000/login?tenant_code=demo` 表示

---

## サブフェーズ 1-5: 認可エンドポイント

**目標:** client/redirect_uri検証、SSO、認可コード発行、PKCE

### 認可リクエスト一時保存方式
ログインページへのリダイレクト時に authorize URL 全体を `redirect_to` パラメータに含める（ステートレス方式）。

### 処理フロー
1. response_type検証（"code"のみ）
2. client_id検証
3. redirect_uri完全一致検証（MUST: 検証失敗時はリダイレクトしない）
4. scope検証（"openid"必須）
5. PKCE検証（S256のみ、plainはサポートしない）
6. セッション確認（prompt=none → error=login_required, prompt=login → 再認証）
7. 認可コード発行（32バイトランダム → hex 64文字）

### 作成ファイル
| ファイル | 内容 |
|---------|------|
| `internal/oidc/authorize.go` | `GET /{tenant_code}/authorize`（ハンドラ + フローロジック一体） |

### 検証
- セッションあり → redirect_uri に code + state
- セッションなし → ログインページにリダイレクト
- 不正 client_id → 400（リダイレクトしない）
- 未登録 redirect_uri → 400（リダイレクトしない）
- prompt=none + セッションなし → error=login_required

---

## サブフェーズ 1-6: トークンエンドポイント

**目標:** authorization_code grant、クライアント認証、PKCE検証、全トークン生成

### 処理フロー
1. クライアント認証（client_secret_basic / client_secret_post）
2. 認可コード検証（存在、未使用、有効期限、client_id一致、redirect_uri一致）
3. PKCE検証: `BASE64URL(SHA256(code_verifier)) == code_challenge`
4. コード使用済みマーク（二重使用時はMUST拒否 + SHOULD既発行トークン失効）
5. アクセストークン生成（JWT: iss, sub, aud, exp, iat, jti, scope, sid）
6. IDトークン生成（JWT: iss, sub, aud, exp, iat, auth_time, nonce, at_hash）
7. リフレッシュトークン生成（ランダム文字列、SHA-256ハッシュでDB保存）
8. Cache-Control: no-store, Pragma: no-cache（MUST）

### 作成ファイル
| ファイル | 内容 |
|---------|------|
| `internal/jwt/token_service.go` | JWT署名・検証（lestrrat-go/jwx v3） |
| `internal/crypto/pkce.go` | PKCE S256検証 |
| `internal/oidc/token.go` | `POST /{tenant_code}/token`（ルーティング） |
| `internal/oidc/token_authcode.go` | Authorization Code Grant フロー |

### 検証
- 認可コード → トークン交換成功
- IDトークンに全必須クレーム
- JWKSの公開鍵でIDトークン署名検証
- 認可コード二重使用 → invalid_grant
- PKCE検証（正/不正）
- Cache-Control ヘッダー確認

---

## サブフェーズ 1-7: UserInfoエンドポイント

**目標:** アクセストークン検証、スコープベースのクレームフィルタリング

### スコープ → クレーム対応
- `openid` → sub（MUST: 常に返す）
- `profile` → name, updated_at
- `email` → email, email_verified

### 作成ファイル
| ファイル | 内容 |
|---------|------|
| `internal/oidc/userinfo.go` | `GET /{tenant_code}/userinfo`（ハンドラ + ロジック一体） |

### 検証
- Bearer トークンで userinfo 取得
- スコープに応じたクレーム返却
- 無効トークン → 401 + WWW-Authenticate

---

## サブフェーズ 1-8: Refresh Token

**目標:** refresh_token grant、Rotation + Reuse Detection

### Rotation
- 旧RT失効 → 新AT/RT/IDToken発行
- 新RTの parent_id = 旧RTのID

### Reuse Detection（RFC 9700 Section 4.14.2）
- 失効済みRTが再利用された場合 → セッション全体を失効（全関連トークン失効）
- reuse_detected_at を記録

### 検証
- RT-1 → RT-2 → RT-3 の正常ローテーション
- RT-1を再利用 → invalid_grant + セッション全体失効
- RT-2も使えなくなること

---

## サブフェーズ 1-9: トークン失効エンドポイント

**目標:** RFC 7009準拠のトークン失効

### 処理フロー
- token_type_hint で検索順序最適化
- アクセストークン失効: そのトークンのみ
- リフレッシュトークン失効: 紐付くアクセストークンも失効
- 存在しないトークン → 200 OK（MUST: RFC 7009 Section 2.2）
- クライアント認証失敗のみ 401

### 作成ファイル
| ファイル | 内容 |
|---------|------|
| `internal/oidc/revoke.go` | `POST /{tenant_code}/revoke`（ハンドラ + ロジック一体） |

### 検証
- アクセストークン失効 → userinfo で 401
- リフレッシュトークン失効 → refresh で invalid_grant
- 存在しないトークン → 200 OK

---

## Phase 1 完了条件

- [ ] Authorization Code Flow（PKCE付き）が動作する
- [ ] IDトークン・アクセストークン・リフレッシュトークンが発行される
- [ ] Refresh Token Rotation + Reuse Detectionが動作する
- [ ] JWKSエンドポイントで公開鍵が取得できる
- [ ] Discoveryエンドポイントが正しいメタ情報を返す

## 作成ファイル総数: 約50ファイル（バックエンド約45 + フロントエンド2 + 設定3）
