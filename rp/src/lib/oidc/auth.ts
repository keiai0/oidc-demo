import * as client from "openid-client";
import { getOIDCConfig, getOIDCEnv } from "./config";

/**
 * 認可リクエストに必要なパラメータを生成し、認可 URL を返す。
 * PAR (RFC 9126) を使用し、認可パラメータをバックチャネルで事前送信する。
 */
export async function buildLoginUrl(options?: {
  claims?: string;
  /** デモ: false にすると state を生成・送信しない */
  stateEnabled?: boolean;
  /** デモ: false にすると PKCE (code_challenge/code_verifier) を生成・送信しない */
  pkceEnabled?: boolean;
  /** デモ: false にすると nonce を生成・送信しない */
  nonceEnabled?: boolean;
}): Promise<{
  url: URL;
  state: string;
  nonce: string;
  codeVerifier: string;
}> {
  const config = await getOIDCConfig();
  const env = getOIDCEnv();

  const stateEnabled = options?.stateEnabled ?? true;
  const pkceEnabled = options?.pkceEnabled ?? true;
  const nonceEnabled = options?.nonceEnabled ?? true;

  let codeVerifier = "";
  let codeChallenge = "";
  if (pkceEnabled) {
    codeVerifier = client.randomPKCECodeVerifier();
    codeChallenge = await client.calculatePKCECodeChallenge(codeVerifier);
  }

  const state = stateEnabled ? client.randomState() : "";
  const nonce = nonceEnabled ? client.randomNonce() : "";

  const params: Record<string, string> = {
    redirect_uri: env.redirectUri,
    scope: "openid profile email",
  };

  if (nonceEnabled) {
    params.nonce = nonce;
  }

  if (pkceEnabled) {
    params.code_challenge = codeChallenge;
    params.code_challenge_method = "S256";
  }

  if (stateEnabled) {
    params.state = state;
  }

  if (options?.claims) {
    params.claims = options.claims;
  }

  // PAR 対応: Discovery で pushed_authorization_request_endpoint が見つかれば PAR を使用
  let url: URL;
  const metadata = config.serverMetadata();
  if (metadata.pushed_authorization_request_endpoint) {
    url = await client.buildAuthorizationUrlWithPAR(config, params);
  } else {
    url = client.buildAuthorizationUrl(config, params);
  }

  return { url, state, nonce, codeVerifier };
}
