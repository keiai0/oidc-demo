import type { ApiError } from "@/types";

const API_URL =
  process.env.NEXT_PUBLIC_OP_BACKEND_BASE_URL ?? "http://localhost:8080";

/** 管理 API のエラーを表す型付きエラークラス。 */
export class ApiRequestError extends Error {
  constructor(
    public readonly code: string,
    public readonly description: string,
    public readonly status: number,
  ) {
    super(description);
    this.name = "ApiRequestError";
  }
}

/** 未知のエラーから表示用メッセージを取得する。 */
export function getErrorMessage(err: unknown): string {
  if (err instanceof ApiRequestError) return err.description;
  if (err instanceof Error) return err.message;
  return "不明なエラーが発生しました";
}

/** Cookie 認証とエラーハンドリングを含む fetch ラッパー。 */
export async function managementFetch<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const headers: Record<string, string> = {
    ...((options?.headers as Record<string, string>) ?? {}),
  };

  // body がある場合のみ Content-Type を設定
  if (options?.body) {
    headers["Content-Type"] ??= "application/json";
  }

  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    credentials: "include",
    headers,
  });

  if (!res.ok) {
    const body: ApiError = await res.json().catch(() => ({
      error: "unknown",
      error_description: `HTTP ${res.status}`,
    }));
    throw new ApiRequestError(body.error, body.error_description, res.status);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return res.json() as Promise<T>;
}
