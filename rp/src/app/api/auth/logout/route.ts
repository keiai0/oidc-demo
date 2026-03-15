import { NextRequest, NextResponse } from "next/server";
import { getSessionId, clearSessionCookie } from "@/lib/session";
import { revokeSession, getSessionForLogout } from "@/lib/db/queries/session";
import { getOIDCConfig, getOIDCEnv } from "@/lib/oidc/config";
import { randomBytes } from "node:crypto";

export const runtime = "nodejs";

/**
 * POST /api/auth/logout
 *
 * RP-Initiated Logout: RP セッションを失効してから OP の end_session_endpoint にリダイレクトする。
 * 仕様参照: RP-Initiated Logout 1.0
 */
export async function POST(request: NextRequest) {
  const sessionId = await getSessionId();

  let idToken: string | null = null;

  if (sessionId) {
    // ログアウト前に idToken を取得（OP に id_token_hint として渡す）
    const session = await getSessionForLogout(sessionId);
    if (session) {
      idToken = session.idToken;
    }

    // RP セッション失効
    await revokeSession(sessionId);
  }

  // Cookie クリア
  await clearSessionCookie();

  // OP の end_session_endpoint にリダイレクト
  try {
    const config = await getOIDCConfig();
    const metadata = config.serverMetadata();
    const endSessionEndpoint = metadata.end_session_endpoint;

    if (endSessionEndpoint) {
      const env = getOIDCEnv();
      const logoutUrl = new URL(endSessionEndpoint);

      if (idToken) {
        logoutUrl.searchParams.set("id_token_hint", idToken);
      }
      logoutUrl.searchParams.set(
        "post_logout_redirect_uri",
        env.postLogoutRedirectUri,
      );
      // CSRF 対策用の state
      const state = randomBytes(16).toString("hex");
      logoutUrl.searchParams.set("state", state);

      // 303 See Other: POST → GET に切り替えてリダイレクト
      return NextResponse.redirect(logoutUrl.toString(), 303);
    }
  } catch (err) {
    console.error("RP-Initiated Logout: failed to get OIDC config:", err);
  }

  // end_session_endpoint が取得できない場合はローカルログアウトのみ
  return NextResponse.redirect(new URL("/", request.nextUrl.origin), 303);
}
