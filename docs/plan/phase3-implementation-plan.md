# Phase 3: RP（動作検証）実装計画

## Context

Phase 1（OPコア）・Phase 2（OP管理機能）が完了し、Authorization Code Flow（PKCE付き）が動作する状態。Phase 3 では OPに対してOIDCクライアントとして動作するRPを構築し、認証フローを画面上でE2E検証できる状態にする。

5つのサブフェーズに分けて段階的に実装・検証を行う。

**必読仕様:** OIDC Core 1.0（クライアント側の検証要件: Section 3.1.3.7）、RFC 9700（OAuth 2.0 Security BCP）

---

## 依存パッケージ追加（3-1 で実施）

```
openid-client          # OIDC クライアント（Discovery, トークン交換, IDトークン検証）
drizzle-orm            # ORM
drizzle-kit            # マイグレーションツール
postgres               # PostgreSQL ドライバ（drizzle-orm用）
@auth/core             # 不要（openid-client で直接実装）
jose                   # JWT デコード（ダッシュボード表示用）
```

## 環境変数（既存 .env.example に定義済み）

| 変数名 | 値 | 用途 |
|--------|------|------|
| `RP_DATABASE_URL` | `postgres://rp_user:rp_password@postgres:5432/oidc_demo?search_path=rp&sslmode=disable` | RP用DBスキーマ |
| `RP_OIDC_ISSUER` | `http://localhost:8080` | OPのissuer URL |
| `RP_OIDC_CLIENT_ID` | `demo-rp` | OPに登録済みのclient_id |
| `RP_OIDC_CLIENT_SECRET` | `demo-rp-secret` | クライアントシークレット |
| `RP_OIDC_REDIRECT_URI` | `http://localhost:3001/api/auth/callback` | コールバックURL |

### 環境変数追加

- `RP_SESSION_SECRET` - RPセッションCookieの署名キー（32バイト以上のランダム文字列）
- `RP_OIDC_POST_LOGOUT_REDIRECT_URI` - ログアウト後のリダイレクト先（`http://localhost:3001`）
- `RP_OIDC_TENANT_CODE` - OPのテナントコード（`demo`）
- `.env.example`, `docker-compose.yml` に追加

---

## サブフェーズ 3-1: Drizzle ORM・DB基盤

**目標:** RPのDBスキーマ定義、マイグレーション基盤、OIDC設定ヘルパー

### 設計方針
- RPのDBは `rp` スキーマを使用（`rp_user` ロール）
- `op_sub`（OPのユーザーID）をキーとしてRP固有のユーザー情報を管理
- セッションにはトークン群（暗号化保存）を含め、SSR時にアクセストークンを利用可能にする
- openid-client の Discovery で OP のメタ情報を自動取得

### 作成ファイル
| ファイル | 内容 |
|---------|------|
| `src/lib/db/index.ts` | Drizzle クライアント初期化（postgres ドライバ） |
| `src/lib/db/schema.ts` | テーブル定義（users, sessions） |
| `drizzle.config.ts` | Drizzle Kit 設定（rp スキーマ） |
| `src/lib/oidc/config.ts` | openid-client 初期化（Discovery + ClientAuth設定） |
| `src/lib/session/index.ts` | セッション管理ヘルパー（Cookie 操作、署名検証） |
| `package.json` | 依存パッケージ追加 |

### テーブル定義（rp スキーマ）

```sql
-- RP側ユーザー（OPのsubをキーにRP固有情報を管理）
users
├── id: uuid (PK, default gen_random_uuid())
├── op_sub: varchar(255) (unique, NOT NULL)     ← OPのusers.id（subクレーム）
├── email: varchar(255)
├── name: varchar(255)
├── created_at: timestamp (default now())
└── updated_at: timestamp (default now())

-- RPセッション（認証状態 + トークン保持）
sessions
├── id: uuid (PK, default gen_random_uuid())
├── user_id: uuid (FK → users, NOT NULL)
├── op_session_id: varchar(255)                 ← OPのsid（SLO用）
├── access_token: text (NOT NULL)               ← 暗号化保存
├── refresh_token: text                         ← 暗号化保存
├── id_token: text (NOT NULL)                   ← JWT文字列（logout時のid_token_hint用）
├── token_expires_at: timestamp (NOT NULL)       ← アクセストークン有効期限
├── expires_at: timestamp (NOT NULL)            ← セッション自体の有効期限
├── revoked_at: timestamp
├── created_at: timestamp (default now())
└── updated_at: timestamp (default now())
```

### マイグレーション
- `drizzle-kit generate` でマイグレーションSQL生成
- `drizzle-kit migrate` で実行
- `package.json` に `"db:generate"`, `"db:migrate"` スクリプト追加

### 検証
- `docker compose --profile rp up --build` → RPが起動
- DB接続確認（`rp` スキーマに `users`, `sessions` テーブル作成）
- openid-client の Discovery で OP のメタ情報取得確認

---

## サブフェーズ 3-2: 認証フロー（ログイン）

**目標:** PKCE付きAuthorization Code Flowでログイン、セッション確立

### 処理フロー

```
1. ユーザーが /api/auth/login にアクセス
2. RPが以下を生成:
   - state（CSRF対策: 32バイトランダム → hex）
   - nonce（IDトークン検証用: 32バイトランダム → hex）
   - code_verifier（PKCE: 43〜128文字のランダム文字列）
   - code_challenge = BASE64URL(SHA256(code_verifier))
3. state, nonce, code_verifier を一時Cookie（HttpOnly, SameSite=Lax）に保存
4. OPの認可エンドポイントにリダイレクト:
   GET /{tenant_code}/authorize?
     response_type=code
     &client_id=demo-rp
     &redirect_uri=http://localhost:3001/api/auth/callback
     &scope=openid profile email
     &state={state}
     &nonce={nonce}
     &code_challenge={code_challenge}
     &code_challenge_method=S256
```

### 作成ファイル
| ファイル | 内容 |
|---------|------|
| `src/lib/oidc/auth.ts` | PKCE生成、state/nonce生成、認可URL構築 |
| `src/app/api/auth/login/route.ts` | `GET /api/auth/login` - 認可リクエスト開始 |

### セキュリティ要件
- `state` はCookieに保存し、コールバック時に一致検証（CSRF対策）
- `code_verifier` はCookieに保存（サーバーサイドのみ使用、HttpOnly）
- `nonce` はCookieに保存し、IDトークンの`nonce`クレームと一致検証
- 一時Cookieは短い有効期限（5分）を設定

### 検証
- `/api/auth/login` → OPの認可エンドポイントにリダイレクトされること
- リダイレクトURLに `code_challenge`, `state`, `nonce` が含まれること
- 一時Cookieが設定されること

---

## サブフェーズ 3-3: コールバック・IDトークン検証

**目標:** 認可コード → トークン交換、IDトークンの全検証項目実装、セッション確立

### 処理フロー

```
1. OPからコールバック: /api/auth/callback?code={code}&state={state}
2. state検証（Cookieの値と一致）
3. トークン交換:
   POST /{tenant_code}/token
     grant_type=authorization_code
     &code={code}
     &redirect_uri={redirect_uri}
     &code_verifier={code_verifier}     ← Cookieから取得
     + クライアント認証（client_secret_basic）
4. IDトークン検証（OIDC Core Section 3.1.3.7 準拠）
5. RP側ユーザー作成 or 更新（op_subをキーにupsert）
6. RPセッション作成（トークン群をDB保存）
7. セッションCookie設定
8. ダッシュボードにリダイレクト
```

### IDトークン検証手順（OIDC Core Section 3.1.3.7）

openid-client がほとんどの検証を自動で行うが、以下が正しく検証されていることを確認する。

| # | 検証項目 | 仕様参照 |
|---|---------|---------|
| 1 | `iss` が OP の Issuer と完全一致 | Section 3.1.3.7 Step 2 |
| 2 | `aud` に自身の client_id が含まれる | Section 3.1.3.7 Step 3 |
| 3 | `aud` が複数値の場合 `azp` が存在し自身の client_id と一致 | Section 3.1.3.7 Step 4, 5 |
| 4 | JWT署名をJWKSの公開鍵（`kid`一致）で検証（alg: RS256） | Section 3.1.3.7 Step 6 |
| 5 | `alg` が期待するアルゴリズムと一致 | Section 3.1.3.7 Step 7 |
| 6 | `exp` が現在時刻より未来 | Section 3.1.3.7 Step 9 |
| 7 | `iat` が許容範囲内（極端に過去でないこと） | Section 3.1.3.7 Step 10 |
| 8 | `nonce` がセッション保存値と一致（送信した場合） | Section 3.1.3.7 Step 11 |
| 9 | `at_hash` の検証（アクセストークンとの紐付け確認） | Section 3.1.3.6 |

### 作成ファイル
| ファイル | 内容 |
|---------|------|
| `src/app/api/auth/callback/route.ts` | `GET /api/auth/callback` - コールバック処理 |
| `src/lib/oidc/token.ts` | トークン交換・IDトークン検証ヘルパー |
| `src/lib/db/queries/user.ts` | ユーザーのupsert処理（op_subキー） |
| `src/lib/db/queries/session.ts` | セッションCRUD |

### エラーハンドリング
- `state` 不一致 → エラーページ（CSRF攻撃の可能性）
- トークン交換失敗 → エラーページ（`error`, `error_description` 表示）
- IDトークン検証失敗 → エラーページ（検証失敗項目を表示）
- OPからのエラーレスポンス（`?error=login_required` 等）→ エラーページ

### 検証
- ログインフロー全体（login → OP認証 → callback → セッション確立）
- RPの `users` テーブルに `op_sub` でユーザーが作成されること
- RPの `sessions` テーブルにトークン群が保存されること
- IDトークンの全検証項目が正しく動作すること
- 不正な `state` → エラー
- 期限切れ認可コード → エラー

---

## サブフェーズ 3-4: ログアウト・セッション管理

**目標:** RPセッション削除、OPへのRP-Initiated Logout

### ログアウト処理フロー

```
1. ユーザーが POST /api/auth/logout を実行
2. RPセッションをDB上で失効（revoked_at を設定）
3. セッションCookieを削除
4. OPのログアウトエンドポイントにリダイレクト:
   GET /{tenant_code}/logout?
     id_token_hint={id_token}           ← セッションに保存したIDトークンJWT
     &post_logout_redirect_uri={uri}    ← RP_OIDC_POST_LOGOUT_REDIRECT_URI
     &state={state}                     ← ログアウト用CSRF対策
```

### セッション管理
- セッションCookieは `rp_session`（HttpOnly, SameSite=Lax）
- セッションCookieの値はセッションID（UUID）に署名を付与（HMAC-SHA256）
- 開発環境では Secure フラグを外す
- セッション有効期限はアクセストークンの有効期限に連動

### ミドルウェア
- `src/middleware.ts` でセッション検証
- 未認証時のリダイレクト制御

### 作成ファイル
| ファイル | 内容 |
|---------|------|
| `src/app/api/auth/logout/route.ts` | `POST /api/auth/logout` - ログアウト処理 |
| `src/middleware.ts` | 認証ミドルウェア（セッション検証・リダイレクト制御） |
| `src/lib/session/cookie.ts` | Cookie署名・検証ヘルパー |

### 検証
- ログアウト → RPセッション失効 → OPログアウトエンドポイントにリダイレクト
- OPログアウト完了 → `post_logout_redirect_uri` にリダイレクト（RPトップページ）
- セッションCookie削除確認
- 認証ミドルウェアの動作（未認証 → ログインページへリダイレクト）

---

## サブフェーズ 3-5: RP画面

**目標:** ログインページ、認証後ダッシュボード

### 画面一覧

#### ログインページ（`/`）
- 未認証時のランディングページ
- 「OPでログイン」ボタン → `/api/auth/login` にリダイレクト
- シンプルなデザイン（動作検証が目的）

#### ダッシュボード（`/dashboard`）
- 認証済みユーザーのみアクセス可能
- 以下の情報をタブまたはセクションで表示:

| セクション | 表示内容 |
|-----------|---------|
| ユーザー情報 | RPのDBに保存された情報（op_sub, email, name） |
| IDトークン | デコードしたヘッダー・ペイロード（全クレーム一覧）、署名検証ステータス |
| アクセストークン | JWT デコード結果（iss, sub, aud, exp, scope 等） |
| userinfo | OPの `/{tenant_code}/userinfo` エンドポイントからリアルタイム取得した結果 |
| セッション情報 | RPセッションID、OPセッションID（sid）、有効期限 |

#### エラーページ（`/error`）
- 認証エラー時の表示ページ
- エラーコード・エラー説明の表示

### userinfoエンドポイント呼び出し
- ダッシュボード表示時にサーバーサイドでOPの userinfo を呼び出す
- アクセストークンを `Authorization: Bearer {access_token}` で送信
- トークン期限切れの場合はリフレッシュトークンで更新を試行

### 作成ファイル
| ファイル | 内容 |
|---------|------|
| `src/app/page.tsx` | ログインページ（未認証時ランディング） |
| `src/app/dashboard/page.tsx` | ダッシュボード（Server Component） |
| `src/app/dashboard/layout.tsx` | ダッシュボードレイアウト（認証チェック） |
| `src/app/error/page.tsx` | エラーページ |
| `src/components/token-viewer.tsx` | トークンデコード表示コンポーネント（Client Component） |
| `src/components/userinfo-viewer.tsx` | userinfo表示コンポーネント |
| `src/components/session-info.tsx` | セッション情報表示コンポーネント |
| `src/lib/oidc/userinfo.ts` | userinfo取得ヘルパー（アクセストークン付きfetch） |
| `src/app/globals.css` | 最小限のスタイル |

### 検証
- 未認証で `/dashboard` → ログインページにリダイレクト
- ログイン完了 → ダッシュボードに遷移
- IDトークンの全クレームが正しく表示される
- userinfoエンドポイントのレスポンスが正しく表示される
- ログアウトボタン → ログアウトフロー実行 → ログインページに戻る

---

## Phase 3 完了条件

- [ ] RPからOPに対してログインが動作する（Authorization Code Flow + PKCE）
- [ ] IDトークンの全検証項目（iss, aud, exp, iat, nonce, 署名, at_hash）が実装されている
- [ ] RPからOPに対してログアウトが動作する（RP-Initiated Logout）
- [ ] ダッシュボードでIDトークン・アクセストークンのデコード結果を確認できる
- [ ] ダッシュボードでuserinfoエンドポイントのレスポンスを確認できる
- [ ] RPのDBにユーザー情報（op_sub）が保存されている

## 作成ファイル総数: 約20ファイル（API 4 + ライブラリ 8 + 画面 6 + 設定 2）
