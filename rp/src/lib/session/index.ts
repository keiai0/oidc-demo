import { createHmac, timingSafeEqual } from "node:crypto";
import { cookies } from "next/headers";
import { getEnv } from "../env";

const COOKIE_NAME = "rp_session";
const SEPARATOR = ".";

/** セッション ID に HMAC-SHA256 署名を付与する */
function sign(sessionId: string, secret: string): string {
  const signature = createHmac("sha256", secret)
    .update(sessionId)
    .digest("hex");
  return `${sessionId}${SEPARATOR}${signature}`;
}

/** 署名付きセッション ID を検証し、セッション ID を返す */
function verify(signedValue: string, secret: string): string | null {
  const separatorIndex = signedValue.lastIndexOf(SEPARATOR);
  if (separatorIndex === -1) return null;

  const sessionId = signedValue.slice(0, separatorIndex);
  const providedSig = signedValue.slice(separatorIndex + 1);

  const expectedSig = createHmac("sha256", secret)
    .update(sessionId)
    .digest("hex");

  // タイミング攻撃対策
  const a = Buffer.from(providedSig, "hex");
  const b = Buffer.from(expectedSig, "hex");
  if (a.length !== b.length) return null;
  if (!timingSafeEqual(a, b)) return null;

  return sessionId;
}

/** セッション Cookie を設定する */
export async function setSessionCookie(sessionId: string): Promise<void> {
  const env = getEnv();
  const signedValue = sign(sessionId, env.sessionSecret);
  const cookieStore = await cookies();
  cookieStore.set(COOKIE_NAME, signedValue, {
    httpOnly: true,
    sameSite: "lax",
    secure: false, // 開発環境のため false
    path: "/",
    maxAge: 60 * 60 * 24, // 24 時間
  });
}

/** セッション Cookie からセッション ID を取得・検証する */
export async function getSessionId(): Promise<string | null> {
  const env = getEnv();
  const cookieStore = await cookies();
  const cookie = cookieStore.get(COOKIE_NAME);
  if (!cookie?.value) return null;
  return verify(cookie.value, env.sessionSecret);
}

/** セッション Cookie を削除する */
export async function clearSessionCookie(): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.delete(COOKIE_NAME);
}
