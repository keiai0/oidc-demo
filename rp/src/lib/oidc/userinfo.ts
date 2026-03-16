import * as client from "openid-client";
import { getOIDCConfig } from "./config";

/**
 * OP の userinfo エンドポイントからユーザー情報を取得する。
 * DPoP Handle が渡された場合は DPoP proof を自動付与する。
 */
export async function fetchUserInfo(
  accessToken: string,
  expectedSub: string,
  dpopHandle?: client.DPoPHandle,
): Promise<Record<string, unknown>> {
  const config = await getOIDCConfig();

  // openid-client の fetchUserInfo を使用（DPoP proof 自動生成対応）
  const userInfo = await client.fetchUserInfo(
    config,
    accessToken,
    expectedSub,
    dpopHandle ? { DPoP: dpopHandle } : undefined,
  );

  return userInfo as Record<string, unknown>;
}
