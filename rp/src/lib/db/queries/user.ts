import { eq } from "drizzle-orm";
import { getDb } from "../index";
import { users } from "../schema";

interface UpsertUserParams {
  opSub: string;
  email?: string | null;
  name?: string | null;
}

/** OP の sub をキーにユーザーを作成 or 更新する */
export async function upsertUser(params: UpsertUserParams) {
  const db = getDb();

  const existing = await db
    .select()
    .from(users)
    .where(eq(users.opSub, params.opSub))
    .limit(1);

  if (existing.length > 0) {
    const [updated] = await db
      .update(users)
      .set({
        email: params.email ?? existing[0].email,
        name: params.name ?? existing[0].name,
        updatedAt: new Date(),
      })
      .where(eq(users.opSub, params.opSub))
      .returning();
    return updated;
  }

  const [created] = await db
    .insert(users)
    .values({
      opSub: params.opSub,
      email: params.email,
      name: params.name,
    })
    .returning();
  return created;
}
