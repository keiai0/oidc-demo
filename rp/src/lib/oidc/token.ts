import * as client from "openid-client";
import { getOIDCConfig, getOIDCEnv } from "./config";

export interface TokenResult {
  accessToken: string;
  refreshToken: string | undefined;
  idToken: string;
  expiresAt: Date;
  claims: client.IDToken;
}

/** 認可コードをトークンに交換する */
export async function exchangeCode(
  callbackUrl: URL,
  codeVerifier: string,
  expectedState: string,
  expectedNonce: string,
): Promise<TokenResult> {
  const config = await getOIDCConfig();

  const tokens = await client.authorizationCodeGrant(config, callbackUrl, {
    pkceCodeVerifier: codeVerifier,
    expectedState,
    expectedNonce,
    idTokenExpected: true,
  });

  const accessToken = tokens.access_token;
  const refreshToken = tokens.refresh_token;
  const idToken = tokens.id_token!;
  const claims = tokens.claims()!;

  // アクセストークンの有効期限
  const expiresIn = tokens.expires_in ?? 3600;
  const expiresAt = new Date(Date.now() + expiresIn * 1000);

  return { accessToken, refreshToken, idToken, expiresAt, claims };
}
