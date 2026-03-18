# セキュリティ攻撃デモ 実装計画

## Context

Phase 1〜3 で Authorization Code Flow（PKCE付き）が動作する状態。state / nonce / PKCE は OP・RP 両方で実装済み。

本計画では、これら3つのセキュリティ機構が**何から守っているか**を、実際に攻撃を体験することで学べるデモ機能を追加する。

---

## 目的

OIDC/OAuth の認可フローには複数のセキュリティ機構が組み込まれているが、初学者にとって以下の違いは分かりにくい:

- **state**: 何を防いでいるのか？
- **nonce**: state と何が違うのか？
- **PKCE**: これがないと何が起きるのか？

各機構を個別に無効化し、対応する攻撃が成功する様子を見せることで、それぞれの**存在意義**を体感させる。

---

## 前提知識: 3つの機構の整理

| 機構 | 守る対象 | 攻撃名 | 攻撃のタイミング |
|------|----------|--------|------------------|
| **state** | 認可レスポンス（callback） | CSRF（認可コードインジェクション） | 攻撃者が自分の認可コードを被害者に踏ませる |
| **nonce** | ID Token | ID Token リプレイ | 漏洩した ID Token を別セッションで再利用 |
| **PKCE** | 認可コード → トークン交換 | 認可コード横取り | 傍受した認可コードでトークンを取得 |

重要な違い:
- **state** はフロントチャネル（リダイレクト）の保護。RP 側で検証する
- **nonce** はトークンの保護。ID Token 内のクレームとして RP 側で検証する
- **PKCE** はバックチャネル（トークンエンドポイント）の保護。OP 側で検証する

---

## デモ設計

### 設計方針

1. **RP にセキュリティ設定パネルを設ける** — state / nonce / PKCE を個別に ON/OFF できるトグル
2. **OP にもデモモード用の検証スキップエンドポイント（またはフラグ）を設ける** — PKCE 検証の無効化
3. **攻撃シナリオをステップバイステップで UI 上にガイド表示する**
4. **各デモは独立して実行可能** — 1つずつ無効化して影響を確認できる

### 安全性の担保

- デモモードは環境変数 `DEMO_MODE=true` でのみ有効化される
- 本番環境では一切の検証スキップが発生しない
- デモモードが無効な場合、設定パネル自体が表示されない

---

## デモ 1: state 無効化 — CSRF 攻撃

### 攻撃シナリオ

1. 攻撃者が OP で自分のアカウントでログインし、認可コード付き callback URL を取得する
2. 攻撃者はこの URL を被害者に踏ませる（メール、掲示板等で共有）
3. 被害者のブラウザが callback URL にアクセスし、攻撃者のアカウントで RP にログインしてしまう
4. 被害者は自分のアカウントだと思い込み、攻撃者のアカウントに機密情報を入力する

### デモフロー

```
通常フロー:
  RP → 認可リクエスト (state=abc123) → OP → callback?code=xxx&state=abc123 → RP が state 検証 ✓

攻撃フロー (state 無効):
  攻撃者: OP で認可コードを取得 → callback?code=attacker_code を被害者に送る
  被害者: callback?code=attacker_code にアクセス → state 検証なし → 攻撃者アカウントでログイン ✗
```

### 必要な変更

**RP 側:**
- ログイン時に state の生成・Cookie 保存をスキップするオプション
- callback で state の検証をスキップするオプション
- 認可リクエスト URL に state パラメータを含めないオプション

**OP 側:**
- 変更なし（OP は state を透過的にエコーするだけで、検証責任は RP にある）

### UI ガイド表示内容

- state を無効にした状態で、攻撃者の認可コードを含む callback URL を生成
- 「この URL を別のブラウザ/シークレットウィンドウで開いてください」と案内
- 攻撃者アカウントでログインしてしまうことを確認

---

## デモ 2: nonce 無効化 — ID Token リプレイ攻撃

### 攻撃シナリオ

1. 攻撃者が何らかの手段で被害者の ID Token を入手する（ログ漏洩、中間者攻撃等）
2. 攻撃者がこの ID Token を自分のセッションで使い回す
3. nonce 検証がなければ、RP はこの ID Token を正当なものとして受け入れる

### Authorization Code Flow における nonce の位置づけ

Authorization Code Flow ではトークン交換がバックチャネル（サーバー間通信）で行われるため、ID Token がフロントチャネルに露出しにくく、リプレイ攻撃のリスクは Implicit Flow より低い。しかし以下のケースでは依然として有効:

- ログ漏洩による ID Token の流出
- バックチャネル通信の TLS 終端が攻撃された場合
- ID Token をクライアント側にキャッシュ・保存している場合

### デモフロー

Authorization Code Flow では ID Token がフロントチャネルに直接露出しないため、**攻撃をシミュレーションする形式**で実装する:

```
通常フロー:
  RP → トークン交換 → ID Token (nonce=xyz789) → RP が nonce 検証 ✓

シミュレーション (nonce 無効):
  1. 正規フローでログイン → ダッシュボードに ID Token が表示される
  2. デモパネルで「ID Token リプレイ攻撃を試す」ボタンを押す
  3. 取得済みの ID Token を使って、別セッション用の検証エンドポイントに送信
  4. nonce 検証あり → 拒否される / nonce 検証なし → 受け入れられる
```

### 必要な変更

**RP 側:**
- ログイン時に nonce の生成・Cookie 保存をスキップするオプション
- callback でのトークン交換時に nonce の検証をスキップするオプション（openid-client の `expectedNonce` を省略）
- リプレイ攻撃シミュレーション用エンドポイント（`POST /api/demo/replay-idtoken`）
  - 任意の ID Token を受け取り、nonce 検証あり/なしで受け入れ可否を返す

**OP 側:**
- 変更なし（OP は nonce を ID Token に含めるだけで、検証責任は RP にある）

### UI ガイド表示内容

- nonce の役割を説明（「この ID Token は、このセッションのためだけに発行されたことを保証する」）
- nonce あり/なしでリプレイの成否が変わることを視覚的に表示
- Authorization Code Flow では直接的なリスクが低い理由も併記（学習目的）

---

## デモ 3: PKCE 無効化 — 認可コード横取り攻撃

### 攻撃シナリオ

1. RP が認可リクエストを送信し、ユーザーが OP で認証する
2. OP が認可コード付き callback URL をリダイレクトで返す
3. 攻撃者がこの認可コードを傍受する（ネイティブアプリのカスタム URL スキーム横取り、ブラウザ拡張、ログ漏洩等）
4. PKCE がなければ、攻撃者は認可コードだけでトークンエンドポイントからトークンを取得できる

### デモフロー

```
通常フロー:
  RP → 認可リクエスト (code_challenge=hash(verifier)) → OP
  OP → callback?code=xxx → RP
  RP → トークンリクエスト (code=xxx, code_verifier=verifier) → OP が検証 ✓

攻撃フロー (PKCE 無効):
  RP → 認可リクエスト (code_challenge なし) → OP
  OP → callback?code=xxx → RP（攻撃者が code を傍受）
  攻撃者 → トークンリクエスト (code=xxx, code_verifier なし) → OP が検証なしでトークン発行 ✗
```

### 必要な変更

**RP 側:**
- ログイン時に code_verifier / code_challenge の生成をスキップするオプション
- 認可リクエスト URL に code_challenge / code_challenge_method を含めないオプション
- callback でのトークン交換時に code_verifier を送信しないオプション

**OP 側:**
- デモモード時、PKCE 検証をスキップするオプション（client の `require_pkce` を一時的に無効化）
- PKCE パラメータなしでも認可コードを発行するオプション

**攻撃シミュレーション用エンドポイント:**
- `POST /api/demo/steal-code` — 認可コードを入力し、code_verifier なしでトークンエンドポイントに直接リクエストを送る
- PKCE 有効時は失敗、無効時は成功することを表示

### UI ガイド表示内容

- 正規フローでログイン後、「認可コードが発行されました: `xxx`」と表示
- 「攻撃者としてこのコードでトークンを取得してみましょう」と案内
- PKCE あり → `invalid_grant` エラー / PKCE なし → トークン取得成功

---

## 実装スコープ

### RP 変更一覧

| ファイル | 変更内容 |
|---------|----------|
| `rp/src/lib/oidc/auth.ts` | state / nonce / PKCE の生成を条件分岐 |
| `rp/src/app/api/auth/login/route.ts` | デモ設定に応じたパラメータ制御 |
| `rp/src/app/api/auth/callback/route.ts` | デモ設定に応じた検証スキップ |
| `rp/src/app/api/demo/config/route.ts` | **新規** デモ設定の取得・更新 API |
| `rp/src/app/api/demo/steal-code/route.ts` | **新規** PKCE 攻撃シミュレーション |
| `rp/src/app/api/demo/replay-idtoken/route.ts` | **新規** nonce 攻撃シミュレーション |
| `rp/src/app/api/demo/csrf-link/route.ts` | **新規** CSRF 攻撃用 callback URL 生成 |
| `rp/src/components/security-demo-panel.tsx` | **新規** セキュリティ設定パネル + 攻撃ガイド UI |
| `rp/src/app/dashboard/page.tsx` | デモパネルの組み込み |

### OP 変更一覧

| ファイル | 変更内容 |
|---------|----------|
| `op/backend/internal/oidc/authorize.go` | デモモード時の PKCE 必須チェックスキップ |
| `op/backend/internal/oidc/token_authcode.go` | デモモード時の PKCE 検証スキップ |
| `op/backend/cmd/server/main.go` | `DEMO_MODE` 環境変数の読み取り・DI |

### 環境変数追加

| 変数名 | デフォルト | 説明 |
|--------|-----------|------|
| `DEMO_MODE` | `false` | デモ機能の有効化（OP） |
| `NEXT_PUBLIC_DEMO_MODE` | `false` | デモ UI の表示制御（RP） |

---

## サブフェーズ構成

### サブフェーズ 1: デモ基盤

**目標:** デモモードの環境変数・設定パネル・状態管理の基盤を構築する。

- [ ] OP: `DEMO_MODE` 環境変数の導入と DI
- [ ] RP: `NEXT_PUBLIC_DEMO_MODE` 環境変数の導入
- [ ] RP: デモ設定の状態管理（Cookie またはサーバー側セッション）
  - `{ stateEnabled: boolean, nonceEnabled: boolean, pkceEnabled: boolean }`
  - デフォルトは全て `true`（通常動作）
- [ ] RP: `POST /api/demo/config` — デモ設定の更新 API
- [ ] RP: `GET /api/demo/config` — デモ設定の取得 API
- [ ] RP: セキュリティ設定パネル UI（トグルスイッチ 3 つ）
- [ ] `.env.example` の更新

### サブフェーズ 2: state 無効化 — CSRF 攻撃デモ

**目標:** state を無効にして CSRF 攻撃を体験できるようにする。

- [ ] RP: `buildLoginUrl()` で state の生成・付与を条件分岐
- [ ] RP: callback で state 検証のスキップ
- [ ] RP: `POST /api/demo/csrf-link` — 攻撃者視点の callback URL 生成
  - 攻撃者としてログインし、認可コード付き callback URL を返す
- [ ] RP: デモパネルに CSRF 攻撃ガイド UI
  - 攻撃手順の説明
  - 生成した攻撃用 URL の表示
  - 「シークレットウィンドウで開いてください」の案内
- [ ] RP: 攻撃成功/失敗の結果表示

### サブフェーズ 3: PKCE 無効化 — 認可コード横取り攻撃デモ

**目標:** PKCE を無効にして認可コード横取り攻撃を体験できるようにする。

- [ ] OP: デモモード時の PKCE 必須チェックスキップ（`authorize.go`）
- [ ] OP: デモモード時の PKCE 検証スキップ（`token_authcode.go`）
- [ ] RP: `buildLoginUrl()` で code_challenge の生成・付与を条件分岐
- [ ] RP: callback でのトークン交換時に code_verifier 省略を条件分岐
- [ ] RP: `POST /api/demo/steal-code` — 攻撃シミュレーション
  - 認可コードを入力として受け取る
  - code_verifier なしでトークンエンドポイントにリクエスト
  - 成功/失敗の結果を返す
- [ ] RP: デモパネルに PKCE 攻撃ガイド UI
  - フロー中に発行された認可コードの表示
  - 攻撃者として「コードを盗んでトークンを取得する」ボタン
  - PKCE あり/なしでの結果比較

### サブフェーズ 4: nonce 無効化 — ID Token リプレイ攻撃デモ

**目標:** nonce を無効にして ID Token リプレイ攻撃をシミュレーションできるようにする。

- [ ] RP: `buildLoginUrl()` で nonce の生成・付与を条件分岐
- [ ] RP: callback でのトークン交換時に nonce 検証スキップを条件分岐
- [ ] RP: `POST /api/demo/replay-idtoken` — リプレイ攻撃シミュレーション
  - ID Token を入力として受け取る
  - nonce 検証あり: 現在のセッションの nonce と不一致 → 拒否
  - nonce 検証なし: ID Token の署名のみ検証 → 受け入れ
- [ ] RP: デモパネルに nonce 攻撃ガイド UI
  - ダッシュボードに表示されている ID Token をコピー
  - 「この ID Token を別のセッションで使い回してみましょう」と案内
  - Authorization Code Flow でのリスクが低い理由の解説を併記

### サブフェーズ 5: 統合・解説

**目標:** 3 つのデモを統合し、比較解説を追加する。

- [ ] デモパネルに総合比較表を追加（state vs nonce vs PKCE の守備範囲）
- [ ] 各デモの「なぜこれが必要か」「ないとどうなるか」の解説テキスト
- [ ] 全デモのウォークスルーテスト
- [ ] ビルド確認（OP: `go build ./...` / RP: `pnpm build`）

---

## 完了条件

- [ ] state を無効にした状態で CSRF 攻撃が成功することを確認できる
- [ ] PKCE を無効にした状態で認可コード横取りが成功することを確認できる
- [ ] nonce を無効にした状態で ID Token リプレイのシミュレーションが成功することを確認できる
- [ ] 全機能有効時は全攻撃が失敗することを確認できる
- [ ] デモモードが無効な場合、設定パネルが表示されず検証スキップも発生しない
- [ ] OP: `go build ./...` が通る
- [ ] RP: `pnpm build` が通る

---

## 仕様参照

| 機構 | 仕様 |
|------|------|
| state | RFC 6749 Section 10.12 (CSRF Protection) |
| PKCE | RFC 7636 全文 |
| nonce | OIDC Core 1.0 Section 3.1.2.1, 15.5.2 |
| セキュリティ BCP | RFC 9700 Section 4 |
