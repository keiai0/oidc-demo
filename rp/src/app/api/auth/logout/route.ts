import { NextRequest, NextResponse } from "next/server";
import { getSessionId, clearSessionCookie } from "@/lib/session";
import { revokeSession } from "@/lib/db/queries/session";

export const runtime = "nodejs";

export async function POST(request: NextRequest) {
  const sessionId = await getSessionId();

  if (sessionId) {
    await revokeSession(sessionId);
  }

  await clearSessionCookie();

  return NextResponse.redirect(new URL("/", request.nextUrl.origin));
}
