import { NextRequest, NextResponse } from "next/server";
import { cookies } from "next/headers";
import { buildLoginUrl } from "@/lib/oidc/auth";
import { isDemoMode } from "@/lib/env";
import { getDemoConfig } from "@/lib/demo/config";

export const runtime = "nodejs";

const COOKIE_MAX_AGE = 300; // 5 分

export async function GET(request: NextRequest) {
  const claims = request.nextUrl.searchParams.get("claims") ?? undefined;

  // デモモード: デモ設定に応じて state / nonce / PKCE の生成を制御
  const demoConfig = isDemoMode() ? await getDemoConfig() : null;
  const stateEnabled = demoConfig?.stateEnabled ?? true;
  const nonceEnabled = demoConfig?.nonceEnabled ?? true;
  const pkceEnabled = demoConfig?.pkceEnabled ?? true;

  const { url, state, nonce, codeVerifier } = await buildLoginUrl({
    claims,
    stateEnabled,
    nonceEnabled,
    pkceEnabled,
  });

  const cookieStore = await cookies();

  // PKCE・CSRF 検証用の一時 Cookie を設定
  const cookieOptions = {
    httpOnly: true,
    sameSite: "lax" as const,
    secure: false,
    path: "/",
    maxAge: COOKIE_MAX_AGE,
  };

  if (stateEnabled) {
    cookieStore.set("oidc_state", state, cookieOptions);
  }
  if (nonceEnabled) {
    cookieStore.set("oidc_nonce", nonce, cookieOptions);
  }
  if (pkceEnabled) {
    cookieStore.set("oidc_code_verifier", codeVerifier, cookieOptions);
  }

  // claims リクエストをコールバック後に表示するために一時保存
  if (claims) {
    cookieStore.set("oidc_claims_request", claims, cookieOptions);
  }

  return NextResponse.redirect(url.toString());
}
