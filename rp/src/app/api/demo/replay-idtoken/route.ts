import { NextRequest, NextResponse } from "next/server";
import { isDemoMode } from "@/lib/env";
import { getEnv } from "@/lib/env";
import { decodeJwt } from "jose";

export const runtime = "nodejs";

/**
 * POST /api/demo/replay-idtoken — ID Token リプレイ攻撃のシミュレーション
 * 漏洩した ID Token を別セッションで送信し、nonce 検証あり/なしで受け入れ可否を返す。
 */
export async function POST(request: NextRequest) {
  if (!isDemoMode()) {
    return NextResponse.json({ error: "demo mode is disabled" }, { status: 404 });
  }

  const body = await request.json();
  const idToken = body.idToken;
  if (!idToken || typeof idToken !== "string") {
    return NextResponse.json({ error: "idToken is required" }, { status: 400 });
  }

  // ID Token をデコードしてクレームを取得
  let claims: Record<string, unknown>;
  try {
    claims = decodeJwt(idToken) as Record<string, unknown>;
  } catch {
    return NextResponse.json({ error: "invalid_id_token", message: "ID Token のデコードに失敗しました" }, { status: 400 });
  }

  const env = getEnv();
  const opBaseUrl = env.oidc.issuerInternal;

  // OP の JWKS で署名を検証（簡易版: Discovery から jwks_uri を取得して検証）
  const discoveryRes = await fetch(
    `${opBaseUrl}/${env.oidc.tenantCode}/.well-known/openid-configuration`,
  );
  if (!discoveryRes.ok) {
    return NextResponse.json({ error: "failed to fetch discovery" }, { status: 500 });
  }
  const discovery = await discoveryRes.json();
  const jwksUri = (discovery.jwks_uri as string).replace(env.oidc.issuer, opBaseUrl);

  // JWKS を取得して署名検証（jose ライブラリを使用）
  const { createRemoteJWKSet, jwtVerify } = await import("jose");
  const JWKS = createRemoteJWKSet(new URL(jwksUri));

  try {
    await jwtVerify(idToken, JWKS, {
      issuer: env.oidc.issuer + "/" + env.oidc.tenantCode,
      audience: env.oidc.clientId,
    });
  } catch {
    return NextResponse.json({
      accepted: false,
      reason: "signature_invalid",
      message: "ID Token の署名検証に失敗しました",
    });
  }

  // nonce 検証: 攻撃者のセッションには nonce がないため、
  // 「別セッションでのリプレイ」をシミュレーション
  const idTokenNonce = claims.nonce as string | undefined;

  // nonce 検証あり: 攻撃者のセッションの nonce（存在しない）と ID Token の nonce を比較
  // → 別セッションなので必ず不一致
  const nonceCheckEnabled = body.nonceCheckEnabled !== false;

  if (nonceCheckEnabled && idTokenNonce) {
    // 攻撃者のセッションには nonce がない = 不一致
    return NextResponse.json({
      accepted: false,
      reason: "nonce_mismatch",
      message: "nonce が現在のセッションと一致しません。この ID Token は別のセッションで発行されたものです。",
      idTokenNonce,
    });
  }

  // nonce 検証なし or ID Token に nonce がない → 受け入れ
  return NextResponse.json({
    accepted: true,
    message: "ID Token が受け入れられました。攻撃者は被害者になりすましてログインできます。",
    claims: {
      sub: claims.sub,
      email: claims.email,
      name: claims.name,
      nonce: idTokenNonce ?? "(なし)",
    },
  });
}
