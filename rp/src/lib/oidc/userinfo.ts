import { getOIDCConfig } from "./config";
import { getEnv } from "../env";

/** OP の userinfo エンドポイントからユーザー情報を取得する */
export async function fetchUserInfo(
  accessToken: string,
): Promise<Record<string, unknown>> {
  const config = await getOIDCConfig();
  const env = getEnv();

  // Discovery メタデータの URL はブラウザ向け（http://localhost:8080）
  // サーバーサイドから呼ぶ場合は内部 URL（http://op-backend:8080）に書き換える
  let userinfoEndpoint = config.serverMetadata().userinfo_endpoint;
  if (!userinfoEndpoint) {
    throw new Error("UserInfo エンドポイントが Discovery メタデータに含まれていません");
  }

  if (env.oidc.issuer !== env.oidc.issuerInternal) {
    userinfoEndpoint = userinfoEndpoint.replace(
      env.oidc.issuer,
      env.oidc.issuerInternal,
    );
  }

  const response = await fetch(userinfoEndpoint, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  });

  if (!response.ok) {
    throw new Error(
      `UserInfo リクエスト失敗: ${response.status} ${response.statusText}`,
    );
  }

  return response.json();
}
