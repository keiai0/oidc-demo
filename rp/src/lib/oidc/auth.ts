import * as client from "openid-client";
import { getOIDCConfig, getOIDCEnv } from "./config";

/** 認可リクエストに必要なパラメータを生成し、認可 URL を返す */
export async function buildLoginUrl(): Promise<{
  url: URL;
  state: string;
  nonce: string;
  codeVerifier: string;
}> {
  const config = await getOIDCConfig();
  const env = getOIDCEnv();

  const codeVerifier = client.randomPKCECodeVerifier();
  const codeChallenge = await client.calculatePKCECodeChallenge(codeVerifier);
  const state = client.randomState();
  const nonce = client.randomNonce();

  // Discovery で取得したメタデータにテナント付きパスの認可エンドポイントが含まれている
  const url = client.buildAuthorizationUrl(config, {
    redirect_uri: env.redirectUri,
    scope: "openid profile email",
    state,
    nonce,
    code_challenge: codeChallenge,
    code_challenge_method: "S256",
  });

  return { url, state, nonce, codeVerifier };
}
