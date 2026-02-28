import * as client from "openid-client";
import { getEnv } from "../env";

let cachedConfig: client.Configuration | null = null;

/** openid-client の Configuration を取得（Discovery 結果をキャッシュ） */
export async function getOIDCConfig(): Promise<client.Configuration> {
  if (cachedConfig) return cachedConfig;

  const env = getEnv();

  // Docker 環境ではコンテナ間通信用の URL で Discovery を取得する
  const discoveryUrl = new URL(
    `/${env.oidc.tenantCode}/.well-known/openid-configuration`,
    env.oidc.issuerInternal,
  );

  cachedConfig = await client.discovery(
    discoveryUrl,
    env.oidc.clientId,
    env.oidc.clientSecret,
    undefined,
    {
      // 開発環境では HTTP を許可する（openid-client v6 はデフォルトで HTTPS のみ）
      execute: [client.allowInsecureRequests],
    },
  );

  // Docker 環境: openid-client がサーバーサイドで行う HTTP リクエスト（トークン交換等）で
  // Discovery メタデータの URL (http://localhost:8080) を内部 URL (http://op-backend:8080) に書き換える
  const issuerPublic = env.oidc.issuer;
  const issuerInternal = env.oidc.issuerInternal;

  if (issuerPublic !== issuerInternal) {
    cachedConfig[client.customFetch] = (url, options) => {
      const rewrittenUrl = url.replace(issuerPublic, issuerInternal);
      return fetch(rewrittenUrl, options as RequestInit);
    };
  }

  return cachedConfig;
}

/** OIDC 設定の環境変数を取得 */
export function getOIDCEnv() {
  return getEnv().oidc;
}
