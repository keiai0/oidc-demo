import { NextRequest, NextResponse } from "next/server";
import { getEnv } from "@/lib/env";

export const runtime = "nodejs";

/**
 * POST /api/demo/rar
 * Rich Authorization Requests (RFC 9396) デモ API。
 *
 * action="get-config": 認可 URL 構築に必要な設定を返す
 * action="exchange": 認可コードをトークンに交換する
 */
export async function POST(request: NextRequest) {
  const body = await request.json();
  const { action } = body;

  const env = getEnv();
  const baseUrl = env.oidc.issuerInternal;
  const tenantCode = env.oidc.tenantCode;

  try {
    switch (action) {
      case "get-config": {
        // クライアントサイドで認可 URL を組み立てるための設定を返す
        // issuer は公開 URL（ブラウザがリダイレクトするため）
        const issuer = env.oidc.issuer;
        return NextResponse.json({
          authorization_endpoint: `${issuer}/${tenantCode}/authorize`,
          client_id: env.oidc.clientId,
          redirect_uri: new URL("/rar-demo", env.oidc.redirectUri.replace("/api/auth/callback", "")).toString(),
        });
      }

      case "exchange": {
        const { code, redirect_uri } = body;
        if (!code) {
          return NextResponse.json(
            { error: "code is required" },
            { status: 400 },
          );
        }

        // 認可コードをトークンに交換（サーバーサイドで client_secret を使用）
        const tokenEndpoint = `${baseUrl}/${tenantCode}/token`;
        const res = await fetch(tokenEndpoint, {
          method: "POST",
          headers: {
            "Content-Type": "application/x-www-form-urlencoded",
            Authorization: `Basic ${Buffer.from(`${env.oidc.clientId}:${env.oidc.clientSecret}`).toString("base64")}`,
          },
          body: new URLSearchParams({
            grant_type: "authorization_code",
            code,
            redirect_uri: redirect_uri || "",
          }).toString(),
        });

        const data = await res.json();
        if (!res.ok) {
          return NextResponse.json(data, { status: res.status });
        }
        return NextResponse.json(data);
      }

      default:
        return NextResponse.json(
          { error: "unknown action" },
          { status: 400 },
        );
    }
  } catch (e) {
    const message =
      e instanceof Error ? e.message : "RAR デモリクエストに失敗しました";
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
