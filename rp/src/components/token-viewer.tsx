"use client";

import { decodeJwt, decodeProtectedHeader } from "jose";

interface TokenViewerProps {
  title: string;
  token: string;
}

export function TokenViewer({ title, token }: TokenViewerProps) {
  let header: Record<string, unknown> = {};
  let payload: Record<string, unknown> = {};
  let decodeError: string | null = null;

  try {
    header = decodeProtectedHeader(token) as Record<string, unknown>;
    payload = decodeJwt(token) as Record<string, unknown>;
  } catch (e) {
    decodeError = e instanceof Error ? e.message : "デコード失敗";
  }

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <h2 className="text-lg font-bold mb-4">{title}</h2>

      {decodeError ? (
        <p className="text-red-600 text-sm">デコードエラー: {decodeError}</p>
      ) : (
        <div className="space-y-4">
          <div>
            <h3 className="text-sm font-semibold text-gray-500 mb-1">
              ヘッダー
            </h3>
            <pre className="bg-gray-50 rounded p-3 text-xs font-mono overflow-x-auto">
              {JSON.stringify(header, null, 2)}
            </pre>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-gray-500 mb-1">
              ペイロード
            </h3>
            <pre className="bg-gray-50 rounded p-3 text-xs font-mono overflow-x-auto">
              {JSON.stringify(
                payload,
                (key, value) => {
                  // exp, iat 等のタイムスタンプを読みやすく表示
                  if (
                    (key === "exp" || key === "iat" || key === "auth_time") &&
                    typeof value === "number"
                  ) {
                    return `${value} (${new Date(value * 1000).toLocaleString("ja-JP", { timeZone: "Asia/Tokyo" })})`;
                  }
                  return value;
                },
                2,
              )}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
}
