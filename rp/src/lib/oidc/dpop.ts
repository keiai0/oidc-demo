import * as client from "openid-client";
import { getOIDCConfig } from "./config";

/**
 * DPoP 鍵ペアと DPoPHandle を生成する。
 * openid-client v6 の getDPoPHandle を利用し、proof JWT 生成を自動化する。
 *
 * DPoP 対応は OP の Discovery メタデータに dpop_signing_alg_values_supported がある場合に有効。
 */
export async function createDPoPHandle(): Promise<client.DPoPHandle | undefined> {
  const config = await getOIDCConfig();
  const metadata = config.serverMetadata();

  // OP が DPoP をサポートしているか確認
  const supportedAlgs = metadata.dpop_signing_alg_values_supported as
    | string[]
    | undefined;
  if (!supportedAlgs || supportedAlgs.length === 0) {
    return undefined;
  }

  // OP がサポートするアルゴリズムの中から優先順位で選択
  const preferredAlgs = ["ES256", "RS256"];
  const alg = preferredAlgs.find((a) => supportedAlgs.includes(a));
  if (!alg) {
    return undefined;
  }

  const keyPair = await client.randomDPoPKeyPair(alg);
  return client.getDPoPHandle(config, keyPair);
}
