import { NextRequest, NextResponse } from "next/server";
import * as jose from "jose";
import { getOIDCConfig, getOIDCEnv } from "@/lib/oidc/config";
import { revokeByOpSessionId } from "@/lib/db/queries/session";

export const runtime = "nodejs";

const BACKCHANNEL_LOGOUT_EVENT =
  "http://schemas.openid.net/event/backchannel-logout";

/**
 * POST /api/auth/backchannel-logout
 *
 * OP からの Back-Channel Logout 通知を受け取り、対象セッションを失効させる。
 * 仕様参照: OIDC Back-Channel Logout 1.0 Section 2.6 (Logout Token Validation)
 */
export async function POST(request: NextRequest) {
  try {
    // application/x-www-form-urlencoded から logout_token を取得
    const formData = await request.formData();
    const logoutToken = formData.get("logout_token");

    if (!logoutToken || typeof logoutToken !== "string") {
      return NextResponse.json(
        { error: "invalid_request", error_description: "logout_token is required" },
        { status: 400 },
      );
    }

    // OP の JWKS を取得して署名検証
    const config = await getOIDCConfig();
    const metadata = config.serverMetadata();
    const jwksUri = metadata.jwks_uri;

    if (!jwksUri) {
      console.error("backchannel-logout: jwks_uri not found in metadata");
      return NextResponse.json(
        { error: "server_error" },
        { status: 500 },
      );
    }

    const JWKS = jose.createRemoteJWKSet(new URL(jwksUri));
    const env = getOIDCEnv();

    // logout_token の署名検証 + クレーム検証
    const { payload } = await jose.jwtVerify(logoutToken, JWKS, {
      issuer: `${env.issuer}/${env.tenantCode}`,
      audience: env.clientId,
    });

    // events クレームの検証 (REQUIRED)
    const events = payload.events as Record<string, unknown> | undefined;
    if (!events || !(BACKCHANNEL_LOGOUT_EVENT in events)) {
      return NextResponse.json(
        { error: "invalid_request", error_description: "missing or invalid events claim" },
        { status: 400 },
      );
    }

    // nonce が含まれていないことを確認 (MUST NOT)
    if ("nonce" in payload) {
      return NextResponse.json(
        { error: "invalid_request", error_description: "logout_token MUST NOT contain nonce" },
        { status: 400 },
      );
    }

    // sub または sid のどちらかが必須 (MUST)
    const sid = payload.sid as string | undefined;
    const sub = payload.sub as string | undefined;

    if (!sid && !sub) {
      return NextResponse.json(
        { error: "invalid_request", error_description: "either sub or sid must be present" },
        { status: 400 },
      );
    }

    // sid がある場合は OP セッション ID で RP セッションを失効
    if (sid) {
      await revokeByOpSessionId(sid);
    }

    // 200 OK で応答 (仕様: 成功を示す)
    return new NextResponse(null, { status: 200 });
  } catch (err) {
    console.error("backchannel-logout: validation error:", err);
    return NextResponse.json(
      { error: "invalid_request", error_description: "logout_token validation failed" },
      { status: 400 },
    );
  }
}
