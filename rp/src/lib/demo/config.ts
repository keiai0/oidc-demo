import { cookies } from "next/headers";

export type DemoConfig = {
  stateEnabled: boolean;
  nonceEnabled: boolean;
  pkceEnabled: boolean;
};

const DEFAULT_CONFIG: DemoConfig = {
  stateEnabled: true,
  nonceEnabled: true,
  pkceEnabled: true,
};

const COOKIE_NAME = "demo_config";
const COOKIE_MAX_AGE = 86400; // 24 時間

/** サーバー側でデモ設定を Cookie から読み取る。未設定時はデフォルト（全て有効）を返す。 */
export async function getDemoConfig(): Promise<DemoConfig> {
  const cookieStore = await cookies();
  const raw = cookieStore.get(COOKIE_NAME)?.value;
  if (!raw) return { ...DEFAULT_CONFIG };

  try {
    const parsed = JSON.parse(raw);
    return {
      stateEnabled: typeof parsed.stateEnabled === "boolean" ? parsed.stateEnabled : true,
      nonceEnabled: typeof parsed.nonceEnabled === "boolean" ? parsed.nonceEnabled : true,
      pkceEnabled: typeof parsed.pkceEnabled === "boolean" ? parsed.pkceEnabled : true,
    };
  } catch {
    return { ...DEFAULT_CONFIG };
  }
}

/** サーバー側でデモ設定を Cookie に書き込む。 */
export async function setDemoConfig(config: DemoConfig): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.set(COOKIE_NAME, JSON.stringify(config), {
    httpOnly: false, // フロントエンドからも読み取る（設定パネル表示用）
    sameSite: "lax",
    secure: false,
    path: "/",
    maxAge: COOKIE_MAX_AGE,
  });
}
