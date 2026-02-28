import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import { buildLoginUrl } from "@/lib/oidc/auth";

export const runtime = "nodejs";

const COOKIE_MAX_AGE = 300; // 5 分

export async function GET() {
  const { url, state, nonce, codeVerifier } = await buildLoginUrl();

  const cookieStore = await cookies();

  // PKCE・CSRF 検証用の一時 Cookie を設定
  const cookieOptions = {
    httpOnly: true,
    sameSite: "lax" as const,
    secure: false,
    path: "/",
    maxAge: COOKIE_MAX_AGE,
  };

  cookieStore.set("oidc_state", state, cookieOptions);
  cookieStore.set("oidc_nonce", nonce, cookieOptions);
  cookieStore.set("oidc_code_verifier", codeVerifier, cookieOptions);

  return NextResponse.redirect(url.toString());
}
