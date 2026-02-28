import {
  pgSchema,
  uuid,
  varchar,
  text,
  timestamp,
} from "drizzle-orm/pg-core";

export const rpSchema = pgSchema("rp");

/** RP 側ユーザー（OP の sub をキーに RP 固有情報を管理） */
export const users = rpSchema.table("users", {
  id: uuid("id").primaryKey().defaultRandom(),
  opSub: varchar("op_sub", { length: 255 }).notNull().unique(),
  email: varchar("email", { length: 255 }),
  name: varchar("name", { length: 255 }),
  createdAt: timestamp("created_at", { withTimezone: true })
    .notNull()
    .defaultNow(),
  updatedAt: timestamp("updated_at", { withTimezone: true })
    .notNull()
    .defaultNow(),
});

/** RP セッション（認証状態 + トークン保持） */
export const sessions = rpSchema.table("sessions", {
  id: uuid("id").primaryKey().defaultRandom(),
  userId: uuid("user_id")
    .notNull()
    .references(() => users.id),
  opSessionId: varchar("op_session_id", { length: 255 }),
  accessToken: text("access_token").notNull(),
  refreshToken: text("refresh_token"),
  idToken: text("id_token").notNull(),
  tokenExpiresAt: timestamp("token_expires_at", { withTimezone: true }).notNull(),
  expiresAt: timestamp("expires_at", { withTimezone: true }).notNull(),
  revokedAt: timestamp("revoked_at", { withTimezone: true }),
  createdAt: timestamp("created_at", { withTimezone: true })
    .notNull()
    .defaultNow(),
});
