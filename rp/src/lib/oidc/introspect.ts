import { getOIDCConfig } from "./config";
import { getEnv } from "../env";

export interface IntrospectionResult {
  active: boolean;
  scope?: string;
  client_id?: string;
  sub?: string;
  exp?: number;
  token_type?: string;
  [key: string]: unknown;
}

/**
 * OP の Token Introspection エンドポイント (RFC 7662) を呼び出し、
 * アクセストークンの有効性をサーバーサイドで確認する。
 */
export async function introspectToken(
  accessToken: string,
): Promise<IntrospectionResult> {
  const config = await getOIDCConfig();
  const env = getEnv();

  const metadata = config.serverMetadata();
  let introspectionEndpoint = metadata.introspection_endpoint as
    | string
    | undefined;
  if (!introspectionEndpoint) {
    throw new Error(
      "Introspection エンドポイントが Discovery メタデータに含まれていません",
    );
  }

  // コンテナ間通信用 URL 書き換え
  if (env.oidc.issuer !== env.oidc.issuerInternal) {
    introspectionEndpoint = introspectionEndpoint.replace(
      env.oidc.issuer,
      env.oidc.issuerInternal,
    );
  }

  // クライアント認証 (client_secret_basic)
  const credentials = Buffer.from(
    `${env.oidc.clientId}:${env.oidc.clientSecret}`,
  ).toString("base64");

  const response = await fetch(introspectionEndpoint, {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
      Authorization: `Basic ${credentials}`,
    },
    body: new URLSearchParams({
      token: accessToken,
      token_type_hint: "access_token",
    }),
  });

  if (!response.ok) {
    throw new Error(
      `Introspection リクエスト失敗: ${response.status} ${response.statusText}`,
    );
  }

  return response.json();
}
