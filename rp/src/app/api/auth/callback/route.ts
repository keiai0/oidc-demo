import { NextRequest, NextResponse } from "next/server";
import { cookies } from "next/headers";
import { exchangeCode } from "@/lib/oidc/token";
import { fetchUserInfo } from "@/lib/oidc/userinfo";
import { upsertUser } from "@/lib/db/queries/user";
import { createSession } from "@/lib/db/queries/session";
import { setSessionCookie } from "@/lib/session";

export const runtime = "nodejs";

export async function GET(request: NextRequest) {
  const { searchParams } = request.nextUrl;

  // OP からのエラーレスポンスをチェック
  const error = searchParams.get("error");
  if (error) {
    const errorDescription =
      searchParams.get("error_description") ?? "不明なエラー";
    const errorUrl = new URL("/error", request.nextUrl.origin);
    errorUrl.searchParams.set("error", error);
    errorUrl.searchParams.set("error_description", errorDescription);
    return NextResponse.redirect(errorUrl.toString());
  }

  // 一時 Cookie から検証値を取得
  const cookieStore = await cookies();
  const expectedState = cookieStore.get("oidc_state")?.value;
  const expectedNonce = cookieStore.get("oidc_nonce")?.value;
  const codeVerifier = cookieStore.get("oidc_code_verifier")?.value;

  if (!expectedState || !expectedNonce || !codeVerifier) {
    const errorUrl = new URL("/error", request.nextUrl.origin);
    errorUrl.searchParams.set("error", "session_expired");
    errorUrl.searchParams.set(
      "error_description",
      "認証セッションが期限切れです。もう一度ログインしてください。",
    );
    return NextResponse.redirect(errorUrl.toString());
  }

  // 一時 Cookie を削除
  cookieStore.delete("oidc_state");
  cookieStore.delete("oidc_nonce");
  cookieStore.delete("oidc_code_verifier");

  try {
    // トークン交換 + ID トークン検証（openid-client が自動検証）
    // openid-client v6 は標準の URL インスタンスを要求する（NextURL は不可）
    const currentUrl = new URL(request.url);
    const tokens = await exchangeCode(
      currentUrl,
      codeVerifier,
      expectedState,
      expectedNonce,
    );

    // UserInfo からユーザー情報を取得（ID トークンに含まれないクレームを補完）
    const userInfo = await fetchUserInfo(tokens.accessToken);

    // RP ユーザー upsert
    const sub = tokens.claims.sub;
    const email =
      typeof userInfo.email === "string" ? userInfo.email : undefined;
    const name =
      typeof userInfo.name === "string" ? userInfo.name : undefined;
    const user = await upsertUser({ opSub: sub, email, name });

    // RP セッション作成
    const sid =
      typeof tokens.claims.sid === "string" ? tokens.claims.sid : undefined;
    const session = await createSession({
      userId: user.id,
      opSessionId: sid,
      accessToken: tokens.accessToken,
      refreshToken: tokens.refreshToken,
      idToken: tokens.idToken,
      tokenExpiresAt: tokens.expiresAt,
      expiresAt: new Date(Date.now() + 24 * 60 * 60 * 1000), // 24 時間
    });

    // セッション Cookie 設定
    await setSessionCookie(session.id);

    return NextResponse.redirect(new URL("/dashboard", request.nextUrl.origin));
  } catch (err) {
    console.error("認証コールバックエラー:", err);
    const errorUrl = new URL("/error", request.nextUrl.origin);
    errorUrl.searchParams.set("error", "callback_failed");
    errorUrl.searchParams.set(
      "error_description",
      err instanceof Error ? err.message : "トークン交換に失敗しました",
    );
    return NextResponse.redirect(errorUrl.toString());
  }
}
