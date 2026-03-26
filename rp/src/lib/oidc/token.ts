import * as client from "openid-client";
import { getOIDCConfig } from "./config";
import { createDPoPHandle } from "./dpop";

export interface TokenResult {
  accessToken: string;
  refreshToken: string | undefined;
  idToken: string;
  expiresAt: Date;
  claims: client.IDToken;
  /** DPoP 使用時のトークン種別 ("DPoP" or "Bearer") */
  tokenType: string;
  /** DPoP Handle（userinfo 等のリソースアクセスに必要） */
  dpopHandle?: client.DPoPHandle;
}

/** 認可コードをトークンに交換する（DPoP 対応） */
export async function exchangeCode(
  callbackUrl: URL,
  codeVerifier: string | undefined,
  expectedState: string | undefined,
  expectedNonce: string | undefined,
): Promise<TokenResult> {
  const config = await getOIDCConfig();

  // DPoP 鍵ペア生成（OP が DPoP をサポートしている場合）
  const dpopHandle = await createDPoPHandle();

  const grantOptions: Record<string, unknown> = {
    idTokenExpected: true,
  };
  // nonce: undefined の場合は検証をスキップ（デモモードで nonce 無効化時）
  if (expectedNonce !== undefined) {
    grantOptions.expectedNonce = expectedNonce;
  }
  // PKCE: codeVerifier が undefined の場合はスキップ（デモモードで PKCE 無効化時）
  if (codeVerifier) {
    grantOptions.pkceCodeVerifier = codeVerifier;
  }
  // state が undefined の場合は検証をスキップ（デモモードで state 無効化時）
  if (expectedState !== undefined) {
    grantOptions.expectedState = expectedState;
  }

  const tokens = await client.authorizationCodeGrant(config, callbackUrl,
    grantOptions as Parameters<typeof client.authorizationCodeGrant>[2],
    undefined, dpopHandle ? { DPoP: dpopHandle } : undefined);

  const accessToken = tokens.access_token;
  const refreshToken = tokens.refresh_token;
  const idToken = tokens.id_token!;
  const claims = tokens.claims()!;

  // アクセストークンの有効期限
  const expiresIn = tokens.expires_in ?? 3600;
  const expiresAt = new Date(Date.now() + expiresIn * 1000);

  const tokenType = tokens.token_type ?? "Bearer";

  return {
    accessToken,
    refreshToken,
    idToken,
    expiresAt,
    claims,
    tokenType,
    dpopHandle,
  };
}
