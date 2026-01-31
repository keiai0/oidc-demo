# Phase 2: OP管理機能 実装計画

## Context

Phase 1（OPコア）が完了し、Authorization Code Flow（PKCE付き）が動作する状態。Phase 2 では管理APIと管理UIを実装し、テナント・クライアント（RP）・鍵の管理をUI上から行えるようにする。

5つのサブフェーズに分けて段階的に実装・検証を行う。

---

## 依存パッケージ追加（2-1 で実施）

### バックエンド（Go）
```
なし（Phase 1 で追加済みのパッケージで対応可能）
```

### フロントエンド（OP frontend）
```
@tanstack/react-query     ← サーバー状態管理・キャッシュ
react-hook-form           ← フォームバリデーション
zod                       ← スキーマバリデーション
@hookform/resolvers       ← zod + react-hook-form 連携
```

## 環境変数追加

- `OP_MANAGEMENT_API_KEY` - 管理API認証用APIキー
- `.env.example`, `docker-compose.yml` に追加

---

## サブフェーズ 2-1: 管理API基盤・テナント管理API

**目標:** 管理API認証ミドルウェア、共通エラーレスポンス、テナントCRUD

### 設計方針
- 管理APIは `/management/v1/*` プレフィックスで OIDC エンドポイントと明確に分離
- 認証方式: APIキー（`Authorization: Bearer {api_key}` ヘッダー）
- エラーレスポンス: REST標準 HTTPステータスコード（03-api-design.md Section 7 準拠）
- リクエストバリデーション: 共通バリデーションミドルウェア

### 管理API共通エラーレスポンス形式
```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "The request body is missing required field: name"
  }
}
```

### 作成ファイル
| ファイル | 内容 |
|---------|------|
| `internal/handler/management/middleware.go` | APIキー認証ミドルウェア |
| `internal/handler/management/error.go` | 管理API共通エラーレスポンス |
| `internal/handler/management/request.go` | 共通リクエストバリデーション・ページネーション |
| `internal/handler/management/response.go` | 共通レスポンス構造体（一覧・詳細） |
| `internal/port/repository/tenant_repository.go` | 変更: 一覧取得・作成・更新メソッド追加 |
| `internal/infrastructure/repository/tenant_repository.go` | 変更: 追加メソッドのGORM実装 |
| `internal/usecase/management/tenant_usecase.go` | TenantUsecase（一覧・詳細・作成・更新） |
| `internal/handler/management/tenant_handler.go` | テナント管理ハンドラー |
| `cmd/server/main.go` | 変更: 管理APIグループ・ミドルウェア・ルート登録 |

### テナント管理API
| メソッド | パス | 説明 |
|---------|------|------|
| `GET` | `/management/v1/tenants` | テナント一覧（ページネーション付き） |
| `POST` | `/management/v1/tenants` | テナント作成 |
| `GET` | `/management/v1/tenants/{tenant_id}` | テナント詳細 |
| `PUT` | `/management/v1/tenants/{tenant_id}` | テナント更新 |

### テナント作成リクエスト
```json
{
  "code": "example-corp",
  "name": "Example Corporation",
  "session_lifetime": 86400,
  "auth_code_lifetime": 120,
  "access_token_lifetime": 3600,
  "refresh_token_lifetime": 604800,
  "id_token_lifetime": 3600
}
```

### テナント更新リクエスト
```json
{
  "name": "Example Corporation Updated",
  "session_lifetime": 43200,
  "access_token_lifetime": 1800
}
```

### バリデーション
- `code`: 必須、英数字とハイフンのみ、3〜64文字、ユニーク
- `name`: 必須、1〜256文字
- `*_lifetime`: 正の整数、上限値チェック

### 検証
- APIキーなし → 401 Unauthorized
- 不正APIキー → 401 Unauthorized
- `GET /management/v1/tenants` → テナント一覧（demo テナント含む）
- `POST /management/v1/tenants` → 201 Created
- code重複 → 409 Conflict
- `GET /management/v1/tenants/{id}` → テナント詳細
- 存在しないID → 404 Not Found
- `PUT /management/v1/tenants/{id}` → 200 OK

---

## サブフェーズ 2-2: クライアント管理API

**目標:** クライアント（RP）のCRUD、シークレットローテーション、redirect_uri管理

### 設計方針
- クライアント作成時に `client_id`（ランダム文字列）と `client_secret` を自動生成
- `client_secret` はレスポンスで一度だけ返し、DBにはハッシュのみ保存
- シークレットローテーション: 新シークレット生成 → 旧シークレットハッシュ上書き
- クライアント削除は論理削除（`status: disabled`）

### 作成ファイル
| ファイル | 内容 |
|---------|------|
| `internal/port/repository/client_repository.go` | 変更: 一覧取得・作成・更新・redirect_uri管理メソッド追加 |
| `internal/infrastructure/repository/client_repository.go` | 変更: 追加メソッドのGORM実装 |
| `internal/usecase/management/client_usecase.go` | ClientUsecase |
| `internal/handler/management/client_handler.go` | クライアント管理ハンドラー |
| `internal/handler/management/redirect_uri_handler.go` | redirect_uri管理ハンドラー |

### クライアント管理API
| メソッド | パス | 説明 |
|---------|------|------|
| `GET` | `/management/v1/tenants/{tenant_id}/clients` | クライアント一覧 |
| `POST` | `/management/v1/tenants/{tenant_id}/clients` | クライアント登録 |
| `GET` | `/management/v1/clients/{client_id}` | クライアント詳細 |
| `PUT` | `/management/v1/clients/{client_id}` | クライアント更新 |
| `DELETE` | `/management/v1/clients/{client_id}` | クライアント無効化（論理削除） |
| `PUT` | `/management/v1/clients/{client_id}/secret` | シークレットローテーション |
| `GET` | `/management/v1/clients/{client_id}/redirect-uris` | redirect_uri一覧 |
| `POST` | `/management/v1/clients/{client_id}/redirect-uris` | redirect_uri追加 |
| `DELETE` | `/management/v1/clients/{client_id}/redirect-uris/{id}` | redirect_uri削除 |

### クライアント登録リクエスト
```json
{
  "name": "Demo RP Application",
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "token_endpoint_auth_method": "client_secret_basic",
  "require_pkce": true,
  "redirect_uris": ["http://localhost:3001/api/auth/callback"],
  "post_logout_redirect_uris": ["http://localhost:3001"],
  "frontchannel_logout_uri": null,
  "backchannel_logout_uri": null
}
```

### クライアント登録レスポンス（201 Created）
```json
{
  "id": "uuid",
  "client_id": "generated-client-id",
  "client_secret": "generated-secret-shown-only-once",
  "name": "Demo RP Application",
  "status": "active",
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "token_endpoint_auth_method": "client_secret_basic",
  "require_pkce": true,
  "redirect_uris": [
    {"id": "uuid", "uri": "http://localhost:3001/api/auth/callback"}
  ],
  "post_logout_redirect_uris": [
    {"id": "uuid", "uri": "http://localhost:3001"}
  ],
  "created_at": "2026-01-01T00:00:00Z"
}
```

### シークレットローテーション
- `PUT /management/v1/clients/{client_id}/secret` → 新シークレット生成
- レスポンスに新 `client_secret` を一度だけ返す
- 旧シークレットは即座に無効化

### バリデーション
- `name`: 必須、1〜256文字
- `grant_types`: 必須、`["authorization_code", "refresh_token", "client_credentials"]` のいずれか
- `response_types`: 必須、`["code"]` のみ
- `token_endpoint_auth_method`: 必須、`client_secret_basic` / `client_secret_post` / `none`
- `redirect_uris`: 必須（`token_endpoint_auth_method` が `none` でない場合）、有効なURL形式
- `redirect_uris`: 完全一致検証のため、フラグメント（`#`）を含むURIは拒否

### 検証
- `POST /management/v1/tenants/{tenant_id}/clients` → 201 + client_secret
- `GET /management/v1/tenants/{tenant_id}/clients` → クライアント一覧
- `PUT /management/v1/clients/{client_id}` → 200 OK（name変更等）
- `DELETE /management/v1/clients/{client_id}` → 204 No Content（status: disabled）
- 無効化済みクライアントで認可リクエスト → エラー
- `PUT /management/v1/clients/{client_id}/secret` → 新 client_secret
- 旧 client_secret でトークンリクエスト → invalid_client
- `POST /management/v1/clients/{client_id}/redirect-uris` → redirect_uri追加
- `DELETE /management/v1/clients/{client_id}/redirect-uris/{id}` → redirect_uri削除
- 登録したクライアントで Phase 1 の認証フローが動作すること

---

## サブフェーズ 2-3: 鍵管理API

**目標:** 署名鍵の一覧取得、ローテーション、無効化

### 設計方針
- 鍵ローテーション: 新鍵生成 → `active: true` → 旧鍵の `active: false` + `rotated_at` 設定
- 旧鍵は即座に削除しない（ローテーション中のJWT検証のため、JWKS には引き続き公開）
- 無効化された鍵は JWKS エンドポイントから除外
- ローテーション後、新しいトークンは新鍵で署名される

### 作成ファイル
| ファイル | 内容 |
|---------|------|
| `internal/port/repository/sign_key_repository.go` | 変更: 一覧取得・無効化メソッド追加 |
| `internal/infrastructure/repository/sign_key_repository.go` | 変更: 追加メソッドのGORM実装 |
| `internal/usecase/management/key_usecase.go` | KeyUsecase（一覧・ローテーション・無効化） |
| `internal/handler/management/key_handler.go` | 鍵管理ハンドラー |

### 鍵管理API
| メソッド | パス | 説明 |
|---------|------|------|
| `GET` | `/management/v1/keys` | 鍵一覧（kid, algorithm, active, created_at, rotated_at） |
| `POST` | `/management/v1/keys/rotate` | 鍵ローテーション（新鍵生成 + 旧鍵の非アクティブ化） |
| `DELETE` | `/management/v1/keys/{kid}` | 鍵の無効化（JWKSから除外） |

### 鍵一覧レスポンス
```json
{
  "keys": [
    {
      "kid": "2026-02-24-a1b2c3d4",
      "algorithm": "RS256",
      "active": true,
      "created_at": "2026-02-24T00:00:00Z",
      "rotated_at": null
    },
    {
      "kid": "2026-01-15-e5f6g7h8",
      "algorithm": "RS256",
      "active": false,
      "created_at": "2026-01-15T00:00:00Z",
      "rotated_at": "2026-02-24T00:00:00Z"
    }
  ]
}
```

### ローテーション処理フロー
```
1. 新RSA鍵ペア生成（2048ビット）
2. 秘密鍵をAES-256-GCMで暗号化してDB保存
3. 新鍵を active: true に設定
4. 現在のアクティブ鍵を active: false + rotated_at 設定
5. JWKSエンドポイントは active + 非active（rotated_atあり）の両方を返す
6. 無効化（DELETE）された鍵のみJWKSから除外
```

### バリデーション
- ローテーション: アクティブ鍵が存在しない場合でも新鍵を生成（初期化相当）
- 無効化: アクティブ鍵の無効化時に警告（アクティブ鍵がゼロになる場合はエラー）
- 無効化: 存在しないkid → 404

### 検証
- `GET /management/v1/keys` → 鍵一覧（Phase 1 で生成した鍵を含む）
- `POST /management/v1/keys/rotate` → 201 Created + 新鍵のkid
- ローテーション後 `/jwks` → 新旧両方の鍵を含む
- ローテーション後に発行したトークン → 新鍵のkidで署名されている
- 旧鍵のkidで署名されたトークン → 引き続き検証可能
- `DELETE /management/v1/keys/{kid}` → 204 No Content
- 無効化後 `/jwks` → 当該鍵が除外されている
- 最後のアクティブ鍵の無効化 → 400 Bad Request

---

## サブフェーズ 2-4: インシデント対応API

**目標:** セキュリティインシデント発生時の緊急トークン失効

### 設計方針
- 全トークン失効: 全テナント・全ユーザーの全トークンを失効（最終手段）
- テナントトークン失効: 特定テナントの全トークンを失効
- ユーザートークン失効: 特定ユーザーの全トークンを失効
- 失効対象: access_tokens, refresh_tokens のみ（id_tokens は失効不要）
- セッションも合わせて失効させる

### 作成ファイル
| ファイル | 内容 |
|---------|------|
| `internal/usecase/management/incident_usecase.go` | IncidentUsecase |
| `internal/handler/management/incident_handler.go` | インシデント対応ハンドラー |
| `internal/port/repository/session_repository.go` | 変更: 一括失効メソッド追加 |
| `internal/port/repository/token_repository.go` | 変更: 一括失効メソッド追加 |
| `internal/infrastructure/repository/session_repository.go` | 変更: 一括失効のGORM実装 |
| `internal/infrastructure/repository/access_token_repository.go` | 変更: 一括失効のGORM実装 |
| `internal/infrastructure/repository/refresh_token_repository.go` | 変更: 一括失効のGORM実装 |

### インシデント対応API
| メソッド | パス | 説明 |
|---------|------|------|
| `POST` | `/management/v1/incidents/revoke-all-tokens` | 全トークン失効 |
| `POST` | `/management/v1/incidents/revoke-tenant-tokens` | テナント全トークン失効 |
| `POST` | `/management/v1/incidents/revoke-user-tokens` | ユーザー全トークン失効 |

### 全トークン失効リクエスト
```json
{
  "reason": "Key compromise detected"
}
```

### テナントトークン失効リクエスト
```json
{
  "tenant_id": "uuid",
  "reason": "Tenant security incident"
}
```

### ユーザートークン失効リクエスト
```json
{
  "user_id": "uuid",
  "reason": "Account compromise reported"
}
```

### 処理フロー
```
1. 対象範囲（全体/テナント/ユーザー）のセッションを全て失効（revoked_at 設定）
2. 対象範囲のアクセストークンを全て失効（revoked_at 設定）
3. 対象範囲のリフレッシュトークンを全て失効（revoked_at 設定）
4. 失効件数をレスポンスに含める
```

### レスポンス（200 OK）
```json
{
  "revoked": {
    "sessions": 42,
    "access_tokens": 156,
    "refresh_tokens": 42
  }
}
```

### 検証
- `POST /management/v1/incidents/revoke-all-tokens` → 全トークン失効
- 失効後、既発行アクセストークンで userinfo → 401
- 失効後、既発行リフレッシュトークンで refresh → invalid_grant
- `POST /management/v1/incidents/revoke-tenant-tokens` → テナント内のみ失効
- 他テナントのトークンは有効なまま
- `POST /management/v1/incidents/revoke-user-tokens` → ユーザーのみ失効
- 同テナントの他ユーザーのトークンは有効なまま

---

## サブフェーズ 2-5: 管理UI（OP frontend）

**目標:** テナント・クライアント・鍵・インシデント対応の管理画面

### 設計方針
- OP frontend（Next.js）に管理画面を追加
- 管理API（`/management/v1/*`）をバックエンドとして使用
- APIキーはフロントエンドの環境変数で管理（`NEXT_PUBLIC_MANAGEMENT_API_KEY` は使わず、Next.js の API Route 経由でプロキシ）
- シンプルなテーブル + フォーム UI（CSS はグローバルスタイルまたはCSS Modules）

### 画面一覧
| 画面 | パス | 内容 |
|------|------|------|
| 管理トップ | `/management` | ダッシュボード（テナント数・クライアント数等） |
| テナント一覧 | `/management/tenants` | テナントリスト + 作成ボタン |
| テナント作成 | `/management/tenants/new` | テナント作成フォーム |
| テナント詳細・編集 | `/management/tenants/[id]` | テナント情報表示・編集フォーム |
| クライアント一覧 | `/management/tenants/[id]/clients` | クライアントリスト + 登録ボタン |
| クライアント登録 | `/management/tenants/[id]/clients/new` | クライアント登録フォーム |
| クライアント詳細・編集 | `/management/clients/[id]` | クライアント情報・redirect_uri管理・シークレットローテーション |
| 鍵管理 | `/management/keys` | 鍵一覧 + ローテーション・無効化ボタン |
| インシデント対応 | `/management/incidents` | トークン一括失効（全体/テナント/ユーザー） |

### 作成ファイル
| ファイル | 内容 |
|---------|------|
| `op/frontend/src/lib/api.ts` | 管理API クライアント（fetch ラッパー） |
| `op/frontend/src/app/management/layout.tsx` | 管理画面共通レイアウト（ナビゲーション） |
| `op/frontend/src/app/management/page.tsx` | ダッシュボード |
| `op/frontend/src/app/management/tenants/page.tsx` | テナント一覧 |
| `op/frontend/src/app/management/tenants/new/page.tsx` | テナント作成 |
| `op/frontend/src/app/management/tenants/[id]/page.tsx` | テナント詳細・編集 |
| `op/frontend/src/app/management/tenants/[id]/clients/page.tsx` | クライアント一覧 |
| `op/frontend/src/app/management/tenants/[id]/clients/new/page.tsx` | クライアント登録 |
| `op/frontend/src/app/management/clients/[id]/page.tsx` | クライアント詳細・編集 |
| `op/frontend/src/app/management/keys/page.tsx` | 鍵管理 |
| `op/frontend/src/app/management/incidents/page.tsx` | インシデント対応 |
| `op/frontend/src/app/api/management/[...path]/route.ts` | 管理APIプロキシ（APIキーをサーバー側で付与） |

### API プロキシ設計
フロントエンドからの管理API呼び出しは Next.js API Route 経由でプロキシし、APIキーをサーバー側で付与する。

```
ブラウザ → /api/management/* → Next.js API Route → /management/v1/* (OP Backend)
                                   ↑ APIキーをここで付与
```

### CORS設定
管理UI (localhost:3000) → OP Backend (localhost:8080) への `/management/v1/*` パスへの CORS 許可。Phase 1 で設定済みの `/internal/*` 向け CORS に追加。

### 主要UIコンポーネント
- テナントフォーム: code, name, 各 lifetime 設定の入力・バリデーション
- クライアント登録フォーム: name, grant_types（チェックボックス）, redirect_uris（動的追加/削除）
- クライアント詳細: シークレットローテーションボタン（確認ダイアログ付き）、redirect_uri の追加/削除
- 鍵管理: ローテーションボタン（確認ダイアログ付き）、各鍵の無効化ボタン
- インシデント対応: 失効範囲選択（全体/テナント/ユーザー）+ 実行ボタン（二重確認ダイアログ）

### 検証
- 管理UIからテナント作成 → テナント一覧に表示
- 管理UIからクライアント登録 → client_secret の表示（コピー可能）
- 登録したクライアントで Phase 1 の認証フロー動作確認
- 管理UIからシークレットローテーション → 新シークレット表示
- 管理UIから鍵ローテーション → JWKS 更新確認
- 管理UIからインシデント対応 → トークン失効確認

---

## Phase 2 完了条件

- [ ] 管理APIが全エンドポイントで動作する（APIキー認証付き）
- [ ] 管理UIからテナントの作成・編集ができる
- [ ] 管理UIからクライアント（RP）を登録できる
- [ ] 登録したクライアントで Phase 1 の認証フローが動作する
- [ ] 管理UIから鍵ローテーションができ、JWKSが更新される
- [ ] 管理UIからインシデント対応（トークン一括失効）ができる

## 作成ファイル総数: 約30ファイル（バックエンド約15 + フロントエンド約15）
