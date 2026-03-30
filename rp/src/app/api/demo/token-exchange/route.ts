import { NextRequest, NextResponse } from "next/server";
import { getSessionId } from "@/lib/session";
import { getSessionWithUser } from "@/lib/db/queries/session";
import { performTokenExchange } from "@/lib/oidc/exchange";

export const runtime = "nodejs";

/**
 * POST /api/demo/token-exchange
 * ダッシュボードから Token Exchange を実行するデモ API。
 */
export async function POST(request: NextRequest) {
  const sessionId = await getSessionId();
  if (!sessionId) {
    return NextResponse.json({ error: "unauthorized" }, { status: 401 });
  }

  const result = await getSessionWithUser(sessionId);
  if (!result) {
    return NextResponse.json({ error: "session not found" }, { status: 401 });
  }

  const body = await request.json();
  const mode = body.mode === "delegation" ? "delegation" : "impersonation";

  try {
    const exchangeResult = await performTokenExchange({
      subjectToken: result.session.accessToken,
      mode,
      // Delegation の場合、同じアクセストークンを actor_token としても使用（デモ用）
      actorToken: mode === "delegation" ? result.session.accessToken : undefined,
      audience: body.audience || undefined,
      scope: body.scope || undefined,
    });

    return NextResponse.json(exchangeResult);
  } catch (e) {
    const message = e instanceof Error ? e.message : "Token Exchange に失敗しました";
    return NextResponse.json({ error: message }, { status: 400 });
  }
}
