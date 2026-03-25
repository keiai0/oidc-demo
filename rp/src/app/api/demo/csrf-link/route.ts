import { NextResponse } from "next/server";
import { isDemoMode } from "@/lib/env";
import { getEnv } from "@/lib/env";

export const runtime = "nodejs";

/** POST /api/demo/csrf-link — 攻撃者の認可コード付き callback URL を生成する */
export async function POST() {
  if (!isDemoMode()) {
    return NextResponse.json({ error: "demo mode is disabled" }, { status: 404 });
  }

  const env = getEnv();
  // コンテナ間通信用 URL で OP のデモ API を叩く
  const opBaseUrl = env.oidc.issuerInternal;

  const res = await fetch(`${opBaseUrl}/api/demo/auth-code`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      login_id: "attacker",
      password: "password",
      client_id: env.oidc.clientId,
      redirect_uri: env.oidc.redirectUri,
      scope: "openid profile email",
    }),
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    return NextResponse.json(
      { error: "failed to generate csrf link", detail: err },
      { status: 500 },
    );
  }

  const data = await res.json();
  return NextResponse.json({ callbackUrl: data.callback_url });
}
