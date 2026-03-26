import { NextRequest, NextResponse } from "next/server";
import { cookies } from "next/headers";
import { exchangeCode } from "@/lib/oidc/token";
import { fetchUserInfo } from "@/lib/oidc/userinfo";
import { upsertUser } from "@/lib/db/queries/user";
import { createSession } from "@/lib/db/queries/session";
import { setSessionCookie } from "@/lib/session";
import { isDemoMode } from "@/lib/env";
import { getDemoConfig } from "@/lib/demo/config";

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

  // デモモード: state / nonce / PKCE 検証の有効/無効を判定
  const demoConfig = isDemoMode() ? await getDemoConfig() : null;
  const stateEnabled = demoConfig?.stateEnabled ?? true;
  const nonceEnabled = demoConfig?.nonceEnabled ?? true;
  const pkceEnabled = demoConfig?.pkceEnabled ?? true;

  // 一時 Cookie から検証値を取得
  const cookieStore = await cookies();
  const expectedState = stateEnabled ? cookieStore.get("oidc_state")?.value : undefined;
  const expectedNonce = nonceEnabled ? cookieStore.get("oidc_nonce")?.value : undefined;
  const codeVerifier = pkceEnabled ? cookieStore.get("oidc_code_verifier")?.value : undefined;

  // 各機構が有効な場合は対応する値も必須
  if ((stateEnabled && !expectedState) || (nonceEnabled && !expectedNonce) || (pkceEnabled && !codeVerifier)) {
    const errorUrl = new URL("/error", request.nextUrl.origin);
    errorUrl.searchParams.set("error", "session_expired");
    errorUrl.searchParams.set(
      "error_description",
      "認証セッションが期限切れです。もう一度ログインしてください。",
    );
    return NextResponse.redirect(errorUrl.toString());
  }

  // デモモード: 認可コードをデモパネル表示用に Cookie に保持
  if (isDemoMode()) {
    const authCode = searchParams.get("code");
    if (authCode) {
      cookieStore.set("demo_last_auth_code", authCode, {
        httpOnly: false,
        sameSite: "lax" as const,
        secure: false,
        path: "/",
        maxAge: 300, // 5 分
      });
    }
  }

  // 一時 Cookie を読み取り・削除
  const claimsRequestRaw = cookieStore.get("oidc_claims_request")?.value;
  if (stateEnabled) {
    cookieStore.delete("oidc_state");
  }
  if (nonceEnabled) {
    cookieStore.delete("oidc_nonce");
  }
  if (pkceEnabled) {
    cookieStore.delete("oidc_code_verifier");
  }
  cookieStore.delete("oidc_claims_request");

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
    // DPoP 使用時は dpopHandle を渡して proof JWT を自動付与
    const userInfo = await fetchUserInfo(
      tokens.accessToken,
      tokens.claims.sub,
      tokens.dpopHandle,
    );

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
    // claims リクエスト JSON をパース（存在すれば）
    let claimsRequestJson: Record<string, unknown> | undefined;
    if (claimsRequestRaw) {
      try {
        claimsRequestJson = JSON.parse(claimsRequestRaw);
      } catch {
        // 無効な JSON は無視
      }
    }

    const session = await createSession({
      userId: user.id,
      opSessionId: sid,
      accessToken: tokens.accessToken,
      refreshToken: tokens.refreshToken,
      idToken: tokens.idToken,
      tokenType: tokens.tokenType,
      userinfoJson: userInfo,
      claimsRequestJson,
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
