import { drizzle } from "drizzle-orm/postgres-js";
import postgres from "postgres";
import * as schema from "./schema";

let _db: ReturnType<typeof drizzle<typeof schema>> | null = null;

/** Drizzle クライアントを遅延初期化で取得する（ビルド時のエラーを回避） */
export function getDb() {
  if (!_db) {
    const databaseUrl = process.env.RP_DATABASE_URL;
    if (!databaseUrl) {
      throw new Error("環境変数 RP_DATABASE_URL が設定されていません");
    }
    const client = postgres(databaseUrl);
    _db = drizzle(client, { schema });
  }
  return _db;
}
