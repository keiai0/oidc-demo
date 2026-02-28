import { eq, and, isNull } from "drizzle-orm";
import { getDb } from "../index";
import { sessions, users } from "../schema";
import { encrypt, decrypt } from "../../crypto";
import { getEnv } from "../../env";

interface CreateSessionParams {
  userId: string;
  opSessionId?: string;
  accessToken: string;
  refreshToken?: string;
  idToken: string;
  tokenExpiresAt: Date;
  expiresAt: Date;
}

/** セッションを作成する（トークンは暗号化して保存） */
export async function createSession(params: CreateSessionParams) {
  const encKey = getEnv().tokenEncryptionKey;
  const db = getDb();

  const [session] = await db
    .insert(sessions)
    .values({
      userId: params.userId,
      opSessionId: params.opSessionId,
      accessToken: encrypt(params.accessToken, encKey),
      refreshToken: params.refreshToken
        ? encrypt(params.refreshToken, encKey)
        : null,
      idToken: params.idToken,
      tokenExpiresAt: params.tokenExpiresAt,
      expiresAt: params.expiresAt,
    })
    .returning();
  return session;
}

/** セッション ID から有効なセッションを取得する（トークンは復号） */
export async function getValidSession(sessionId: string) {
  const db = getDb();
  const [session] = await db
    .select()
    .from(sessions)
    .where(and(eq(sessions.id, sessionId), isNull(sessions.revokedAt)))
    .limit(1);

  if (!session) return null;

  // 有効期限チェック
  if (session.expiresAt < new Date()) return null;

  const encKey = getEnv().tokenEncryptionKey;
  return {
    ...session,
    accessToken: decrypt(session.accessToken, encKey),
    refreshToken: session.refreshToken
      ? decrypt(session.refreshToken, encKey)
      : null,
  };
}

/** セッション ID からセッションとユーザー情報を取得する */
export async function getSessionWithUser(sessionId: string) {
  const db = getDb();
  const result = await db
    .select({
      session: sessions,
      user: users,
    })
    .from(sessions)
    .innerJoin(users, eq(sessions.userId, users.id))
    .where(and(eq(sessions.id, sessionId), isNull(sessions.revokedAt)))
    .limit(1);

  if (result.length === 0) return null;

  const { session, user } = result[0];

  // 有効期限チェック
  if (session.expiresAt < new Date()) return null;

  const encKey = getEnv().tokenEncryptionKey;
  return {
    session: {
      ...session,
      accessToken: decrypt(session.accessToken, encKey),
      refreshToken: session.refreshToken
        ? decrypt(session.refreshToken, encKey)
        : null,
    },
    user,
  };
}

/** セッションを失効させる */
export async function revokeSession(sessionId: string) {
  const db = getDb();
  await db
    .update(sessions)
    .set({ revokedAt: new Date() })
    .where(eq(sessions.id, sessionId));
}
