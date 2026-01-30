# 純粋なOIDC認証基盤 設計ガイド

将来、OIDC準拠の認証基盤（OP: OpenID Provider）を新規開発する際の設計の柱となるドキュメント群。

KKPPプロジェクトの反省を踏まえ、「認証基盤として必要なもの」を純化した設計指針をまとめる。

## ドキュメント構成

| ドキュメント | 内容 |
|-------------|------|
| [01-scope.md](./01-scope.md) | スコープ定義：OPの責務境界と設計思想 |
| [02-domain-model.md](./02-domain-model.md) | ドメインモデル・DB設計 |
| [03-api-design.md](./03-api-design.md) | APIインターフェース設計 |

## このガイドの前提

- **対象**: マルチテナント対応のエンタープライズ向けOIDC認証基盤
- **教訓元**: KKPPプロジェクト（認証基盤にRP業務機能が混入した反省）

---

## 参照すべき標準仕様

### 必須（コア仕様）

| 仕様 | 内容 | 読む優先度 |
|------|------|-----------|
| [OAuth 2.0 RFC 6749](https://datatracker.ietf.org/doc/html/rfc6749) | 認可フロー全般、grant_type、トークンエンドポイント | ★★★ 最初に読む |
| [OIDC Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html) | 認証フロー、IDトークン、userinfoエンドポイント、標準クレーム | ★★★ 最初に読む |
| [JWT RFC 7519](https://datatracker.ietf.org/doc/html/rfc7519) | JWTの構造・クレーム定義 | ★★★ JWT実装前に必読 |
| [JWK RFC 7517](https://datatracker.ietf.org/doc/html/rfc7517) | JWKSエンドポイントの鍵フォーマット | ★★★ JWT実装前に必読 |
| [JWA RFC 7518](https://datatracker.ietf.org/doc/html/rfc7518) | RS256等の署名アルゴリズム定義 | ★★★ JWT実装前に必読 |
| [OAuth 2.0 Bearer Token RFC 6750](https://datatracker.ietf.org/doc/html/rfc6750) | アクセストークンの使い方（Authorizationヘッダー等） | ★★☆ |

### 必須（エンドポイント・拡張）

| 仕様 | 内容 | 読む優先度 |
|------|------|-----------|
| [OIDC Discovery 1.0](https://openid.net/specs/openid-connect-discovery-1_0.html) | `/.well-known/openid-configuration`の仕様、issuerの検証ルール | ★★★ Discovery実装前に必読 |
| [PKCE RFC 7636](https://datatracker.ietf.org/doc/html/rfc7636) | code_challenge、S256メソッド | ★★★ PKCE実装前に必読 |
| [Token Revocation RFC 7009](https://datatracker.ietf.org/doc/html/rfc7009) | `/revoke`エンドポイントの仕様 | ★★☆ |

### 必須（SLO）

| 仕様 | 内容 | 読む優先度 |
|------|------|-----------|
| [RP-Initiated Logout 1.0](https://openid.net/specs/openid-connect-rpinitiated-1_0.html) | `/logout`エンドポイント、`id_token_hint`、`post_logout_redirect_uri` | ★★★ SLO実装前に必読 |
| [Front-Channel Logout 1.0](https://openid.net/specs/openid-connect-frontchannel-1_0.html) | iframeによるRP通知、`iss`/`sid`パラメータのルール | ★★★ SLO実装前に必読 |
| [Back-Channel Logout 1.0](https://openid.net/specs/openid-connect-backchannel-1_0.html) | `logout_token`の構造・クレーム要件、サーバー間通知 | ★★★ SLO実装前に必読 |

### 必須（セキュリティ）

| 仕様 | 内容 | 読む優先度 |
|------|------|-----------|
| [OAuth 2.0 Security BCP RFC 9700](https://datatracker.ietf.org/doc/html/rfc9700) | Refresh Token Rotation、Reuse Detection、redirect_uri完全一致等の現代的なベストプラクティス | ★★★ 設計初期に通読 |

### 状況に応じて参照

| 仕様 | 参照すべき状況 |
|------|--------------|
| [OIDC Session Management 1.0](https://openid.net/specs/openid-connect-session-1_0.html) | iframeを使ったセッション状態の定期確認を実装する場合 |
| [Token Introspection RFC 7662](https://datatracker.ietf.org/doc/html/rfc7662) | RPがOPにトークンの有効性を問い合わせる`/introspect`エンドポイントを実装する場合 |
| [WebAuthn Level 2](https://www.w3.org/TR/webauthn-2/) | パスキー認証を実装する場合 |
| [mTLS RFC 8705](https://datatracker.ietf.org/doc/html/rfc8705) | Refresh Tokenをクライアント証明書に紐付けるsender-constrainingを実装する場合 |
