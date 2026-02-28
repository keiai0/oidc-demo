import { NextRequest, NextResponse } from "next/server";

const COOKIE_NAME = "rp_session";
const SEPARATOR = ".";

/** Web Crypto API を使って HMAC-SHA256 署名を検証する（Edge Runtime 対応） */
async function verifySignature(
  signedValue: string,
  secret: string,
): Promise<boolean> {
  const separatorIndex = signedValue.lastIndexOf(SEPARATOR);
  if (separatorIndex === -1) return false;

  const sessionId = signedValue.slice(0, separatorIndex);
  const providedSig = signedValue.slice(separatorIndex + 1);

  const encoder = new TextEncoder();
  const key = await crypto.subtle.importKey(
    "raw",
    encoder.encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );

  const signatureBuffer = await crypto.subtle.sign(
    "HMAC",
    key,
    encoder.encode(sessionId),
  );

  const expectedSig = Array.from(new Uint8Array(signatureBuffer))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");

  // 定数時間比較（簡易版: 長さが異なればすぐ false）
  if (providedSig.length !== expectedSig.length) return false;
  let result = 0;
  for (let i = 0; i < providedSig.length; i++) {
    result |= providedSig.charCodeAt(i) ^ expectedSig.charCodeAt(i);
  }
  return result === 0;
}

export async function middleware(request: NextRequest) {
  const cookie = request.cookies.get(COOKIE_NAME);
  const secret = process.env.RP_SESSION_SECRET;

  if (!cookie?.value || !secret) {
    return NextResponse.redirect(new URL("/", request.url));
  }

  const valid = await verifySignature(cookie.value, secret);
  if (!valid) {
    return NextResponse.redirect(new URL("/", request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/dashboard/:path*"],
};
