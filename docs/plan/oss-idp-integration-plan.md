# OSS IdP 統合計画

## 目的

自作 OP（`op/backend`）と並列で、既存の OSS 製 IdP を docker-compose 上で動かせるようにする。RP（`rp/`）から接続先を切り替えて、以下の学習効果を得ることを狙う。

1. **RFC 準拠挙動の答え合わせ** — 自作 OP の挙動を本家実装と比較し、仕様解釈のズレを検出する
2. **実装パターンの参照** — OSS 実装のコードリーディングから、自作 OP に取り込むべき設計を抽出する
3. **実務互換性の検証** — 実運用で遭遇する IdP に RP が接続できることを確認する

**非目的**: 自作 OP を OSS IdP で置き換えることは目的としない。自作 OP の学習・実装は継続する。

## 方針

- **並列稼働**: 自作 OP（:8080）と OSS IdP を別ポートで同時起動可能にする
- **profile 分離**: `docker-compose.yml` の profiles で OSS IdP を opt-in 起動する（`--profile hydra` 等）
  - 既存 `op` / `rp` / `all` profile には影響を与えない
- **RP 側の切替**: RP のログイン画面から接続先 IdP を選択できるようにする（複数 issuer 設定を保持）
- **シークレット管理**: CLAUDE.md 方針に従い `.env` で管理し、`.env.example` を更新する

## 対象 IdP（優先順位順）

### Phase A: Ory Hydra（最優先）

**選定理由**
- OAuth2/OIDC 仕様に特化した純粋実装で、自作 OP と 1:1 で挙動比較しやすい
- Go 製で `docs/research/` に既にコード調査済み
- CLAUDE.md に記載の「構造化 OIDC エラー型（Fluent API）」の参考元

**構成**
- `oryd/hydra:v2.2`（public: 4444, admin: 4445）
- Login/Consent UI: まずは公式サンプル `oryd/hydra-login-consent-node` を利用（:3002）
- DB: 既存 PostgreSQL に `hydra` スキーマを追加、または Hydra 専用 DB コンテナを立てる（検討事項）

**学習ポイント**
- Admin API からの client 登録フロー
- Login/Consent 外部委譲の設計思想
- `/oauth2/auth`, `/oauth2/token`, `/userinfo`, `/.well-known/openid-configuration` の挙動比較
- エラーレスポンスの差分（RFC 6749 Section 5.2 準拠度）

### Phase B: Keycloak

**選定理由**
- 実務で遭遇する確率が最も高い
- 管理 UI・フェデレーション・SAML など広範な機能を持ち、RP 側の互換性検証に最適

**構成**
- `quay.io/keycloak/keycloak:latest`（:8180）
- realm / client を起動時に import（`realm-export.json`）
- DB: Keycloak 専用（H2 開発モード、または PostgreSQL）

**学習ポイント**
- realm・client・role の概念整理
- ID Token / Access Token のクレーム構造比較
- 管理 UI から発行されるトークンの検証

### Phase C: Dex（任意）

**選定理由**
- Go コードリーディング用。CLAUDE.md で既に参考パターン抽出済み（Conformance Test、コンパイル時 interface チェック、time 関数注入、slog）
- 軽量で読みやすい

**構成**
- `ghcr.io/dexidp/dex:latest`（:5556）
- `config.yaml` で static clients・static passwords を定義

**優先度**: 低。Phase A/B が落ち着いた後に追加検討する。

## 実装ステップ

### Step 1: Hydra の docker-compose 統合

- [ ] `docker-compose.yml` に `hydra`, `hydra-migrate`, `hydra-consent` サービスを追加
- [ ] profile `hydra` / `all` に所属させる
- [ ] 環境変数を `.env` に追加し、`.env.example` を更新
  - `HYDRA_SECRETS_SYSTEM`, `HYDRA_DSN`, `HYDRA_URLS_SELF_ISSUER` 等
- [ ] Hydra 用 DB（PostgreSQL スキーマ or 独立コンテナ）を決定して構築
- [ ] `docker compose --profile hydra up -d` で起動確認
- [ ] Discovery エンドポイント `http://localhost:4444/.well-known/openid-configuration` の取得確認

### Step 2: Hydra client 登録と動作確認

- [ ] Admin API（:4445）経由で RP 用 client を登録するスクリプトを作成（`scripts/hydra-bootstrap.sh` 等）
- [ ] curl で認可コードフローを一通り実行（認可リクエスト → Login/Consent → token 取得 → userinfo）
- [ ] 自作 OP との挙動差分をメモ（`docs/research/` に追記するか検討）

### Step 3: RP の複数 issuer 対応

- [ ] RP の環境変数を複数 IdP 設定に拡張（`OIDC_ISSUER_SELF`, `OIDC_ISSUER_HYDRA` 等）
- [ ] ログイン画面に IdP 選択 UI を追加（「自作 OP でログイン」「Hydra でログイン」）
- [ ] 選択した issuer に応じて認可リクエストの送信先を切り替える
- [ ] Callback 処理で issuer を識別し、対応する token endpoint / JWKS を使う
- [ ] セッションに「どの IdP でログインしたか」を保持する

### Step 4: Keycloak の docker-compose 統合

- [ ] `docker-compose.yml` に `keycloak` サービスを追加（profile `keycloak` / `all`）
- [ ] 起動時 import 用の `realm-export.json` を作成（RP 用 client を含む）
- [ ] `.env` / `.env.example` 更新
- [ ] `docker compose --profile keycloak up -d` で起動確認
- [ ] Discovery エンドポイント `http://localhost:8180/realms/<realm>/.well-known/openid-configuration` の確認

### Step 5: RP から Keycloak 接続確認

- [ ] RP の複数 issuer 設定に Keycloak を追加
- [ ] ログイン画面に Keycloak 選択肢を追加
- [ ] 認可コードフローの一通り確認
- [ ] 自作 OP / Hydra / Keycloak の3者でクレーム構造を比較（ドキュメント化）

### Step 6（任意）: Dex の統合

- [ ] `docker-compose.yml` に `dex` サービスを追加（profile `dex` / `all`）
- [ ] `config.yaml` 作成
- [ ] 動作確認

## 検討事項

- **DB 分離方針**: Hydra / Keycloak は既存 PostgreSQL にスキーマを切るか、専用コンテナを立てるか
  - 既存スキーマ方式: リソース節約、起動が早い
  - 専用コンテナ方式: 本家ドキュメントと構成が一致し、設定が簡素
  - → 学習用途なので **専用コンテナ方式** を第一候補とする（推奨）
- **Hydra Login/Consent UI 自作化**: 公式サンプルで動作確認後、自作 OP の Login/Consent 画面を Hydra 用にも使う統合案は将来検討
- **issuer 切替の粒度**: RP のセッションに IdP 識別子を持たせる必要があるため、既存セッション設計の見直しが発生する可能性あり

## 完了条件

- `docker compose --profile hydra up -d` / `--profile keycloak up -d` で各 IdP が起動する
- RP のログイン画面から自作 OP / Hydra / Keycloak を選択してログインできる
- 自作 OP と Hydra/Keycloak の挙動差分がドキュメント化されている
- `.env.example` が最新化されている

## 参考

- Ory Hydra: https://www.ory.sh/docs/hydra
- Keycloak: https://www.keycloak.org/documentation
- Dex: https://dexidp.io/docs/
- 既存調査: `docs/research/hydra/`, `docs/research/dex/`
