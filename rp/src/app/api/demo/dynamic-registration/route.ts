import { NextRequest, NextResponse } from "next/server";
import { getEnv } from "@/lib/env";

export const runtime = "nodejs";

/**
 * POST /api/demo/dynamic-registration
 * Dynamic Client Registration (RFC 7591) のデモ API。
 * RP サーバーサイドから OP の registration endpoint を呼び出す。
 */
export async function POST(request: NextRequest) {
  const body = await request.json();
  const { action, initialAccessToken, registrationAccessToken, clientId, metadata } = body;

  const env = getEnv();
  const baseUrl = env.oidc.issuerInternal;

  try {
    switch (action) {
      case "register": {
        // POST /{tenant_code}/register
        const res = await fetch(`${baseUrl}/register`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${initialAccessToken}`,
          },
          body: JSON.stringify(metadata),
        });
        const data = await res.json();
        if (!res.ok) {
          return NextResponse.json(data, { status: res.status });
        }
        return NextResponse.json(data, { status: 201 });
      }

      case "get": {
        // GET /{tenant_code}/register/{client_id}
        const res = await fetch(`${baseUrl}/register/${clientId}`, {
          headers: {
            Authorization: `Bearer ${registrationAccessToken}`,
          },
        });
        const data = await res.json();
        if (!res.ok) {
          return NextResponse.json(data, { status: res.status });
        }
        return NextResponse.json(data);
      }

      case "update": {
        // PUT /{tenant_code}/register/{client_id}
        const res = await fetch(`${baseUrl}/register/${clientId}`, {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${registrationAccessToken}`,
          },
          body: JSON.stringify(metadata),
        });
        const data = await res.json();
        if (!res.ok) {
          return NextResponse.json(data, { status: res.status });
        }
        return NextResponse.json(data);
      }

      case "delete": {
        // DELETE /{tenant_code}/register/{client_id}
        const res = await fetch(`${baseUrl}/register/${clientId}`, {
          method: "DELETE",
          headers: {
            Authorization: `Bearer ${registrationAccessToken}`,
          },
        });
        if (res.status === 204) {
          return NextResponse.json({ message: "client deleted" });
        }
        const data = await res.json();
        return NextResponse.json(data, { status: res.status });
      }

      default:
        return NextResponse.json(
          { error: "unknown action" },
          { status: 400 },
        );
    }
  } catch (e) {
    const message =
      e instanceof Error ? e.message : "Dynamic Registration に失敗しました";
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
