import { NextRequest, NextResponse } from "next/server";
import { isDemoMode } from "@/lib/env";
import { getDemoConfig, setDemoConfig, type DemoConfig } from "@/lib/demo/config";

export const runtime = "nodejs";

export async function GET() {
  if (!isDemoMode()) {
    return NextResponse.json({ error: "demo mode is disabled" }, { status: 404 });
  }

  const config = await getDemoConfig();
  return NextResponse.json(config);
}

export async function POST(request: NextRequest) {
  if (!isDemoMode()) {
    return NextResponse.json({ error: "demo mode is disabled" }, { status: 404 });
  }

  const body = await request.json();

  const config: DemoConfig = {
    stateEnabled: typeof body.stateEnabled === "boolean" ? body.stateEnabled : true,
    nonceEnabled: typeof body.nonceEnabled === "boolean" ? body.nonceEnabled : true,
    pkceEnabled: typeof body.pkceEnabled === "boolean" ? body.pkceEnabled : true,
  };

  await setDemoConfig(config);
  return NextResponse.json(config);
}
