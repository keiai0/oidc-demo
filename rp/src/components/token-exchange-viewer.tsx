"use client";

import { useState } from "react";
import { decodeJwt } from "jose";

interface TokenExchangeViewerProps {
  originalToken: string;
}

interface ExchangeResult {
  access_token: string;
  issued_token_type: string;
  token_type: string;
  expires_in: number;
  scope?: string;
}

export function TokenExchangeViewer({
  originalToken,
}: TokenExchangeViewerProps) {
  const [mode, setMode] = useState<"impersonation" | "delegation">(
    "impersonation",
  );
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<ExchangeResult | null>(null);

  const handleExchange = async () => {
    setLoading(true);
    setError(null);
    setResult(null);

    try {
      const res = await fetch("/api/demo/token-exchange", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ mode }),
      });

      const data = await res.json();
      if (!res.ok) {
        setError(data.error ?? `HTTP ${res.status}`);
        return;
      }
      setResult(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "リクエストに失敗しました");
    } finally {
      setLoading(false);
    }
  };

  let originalPayload: Record<string, unknown> = {};
  let exchangedPayload: Record<string, unknown> = {};

  try {
    originalPayload = decodeJwt(originalToken) as Record<string, unknown>;
  } catch {
    // ignore
  }
  if (result?.access_token) {
    try {
      exchangedPayload = decodeJwt(
        result.access_token,
      ) as Record<string, unknown>;
    } catch {
      // ignore
    }
  }

  const formatPayload = (payload: Record<string, unknown>) =>
    JSON.stringify(
      payload,
      (key, value) => {
        if (
          (key === "exp" || key === "iat") &&
          typeof value === "number"
        ) {
          return `${value} (${new Date(value * 1000).toLocaleString("ja-JP", { timeZone: "Asia/Tokyo" })})`;
        }
        return value;
      },
      2,
    );

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <h2 className="text-lg font-bold mb-4">
        Token Exchange (RFC 8693) デモ
      </h2>

      <div className="space-y-4">
        {/* Mode Selection */}
        <div>
          <label className="block text-sm font-semibold text-gray-600 mb-2">
            交換モード
          </label>
          <div className="flex gap-4">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="radio"
                name="mode"
                value="impersonation"
                checked={mode === "impersonation"}
                onChange={() => setMode("impersonation")}
              />
              Impersonation（なりすまし）
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="radio"
                name="mode"
                value="delegation"
                checked={mode === "delegation"}
                onChange={() => setMode("delegation")}
              />
              Delegation（委任）
            </label>
          </div>
          <p className="text-xs text-gray-500 mt-1">
            {mode === "impersonation"
              ? "Impersonation: 発行されるトークンの sub は元のユーザー。actor は見えない。"
              : "Delegation: act クレームで委任チェーンを明示する。sub は元のユーザー。"}
          </p>
        </div>

        {/* Execute Button */}
        <button
          onClick={handleExchange}
          disabled={loading}
          className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700 disabled:opacity-50"
        >
          {loading ? "交換中..." : "Token Exchange 実行"}
        </button>

        {error && (
          <div className="bg-red-50 border border-red-200 rounded p-3 text-sm text-red-700">
            {error}
          </div>
        )}

        {/* Result */}
        {result && (
          <div className="space-y-4">
            {/* Response Info */}
            <div>
              <h3 className="text-sm font-semibold text-gray-600 mb-1">
                レスポンス
              </h3>
              <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs bg-gray-50 rounded p-3">
                <dt className="text-gray-500">issued_token_type</dt>
                <dd className="font-mono">{result.issued_token_type}</dd>
                <dt className="text-gray-500">token_type</dt>
                <dd>{result.token_type}</dd>
                <dt className="text-gray-500">expires_in</dt>
                <dd>{result.expires_in}秒</dd>
                <dt className="text-gray-500">scope</dt>
                <dd>{result.scope || "(なし)"}</dd>
              </dl>
            </div>

            {/* Token Comparison */}
            <div>
              <h3 className="text-sm font-semibold text-gray-600 mb-2">
                トークン比較（交換前 → 交換後）
              </h3>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <h4 className="text-xs font-medium text-gray-500 mb-1">
                    交換前（元のアクセストークン）
                  </h4>
                  <pre className="bg-gray-50 rounded p-3 text-xs font-mono overflow-x-auto max-h-80 overflow-y-auto">
                    {formatPayload(originalPayload)}
                  </pre>
                </div>
                <div>
                  <h4 className="text-xs font-medium text-gray-500 mb-1">
                    交換後（新しいアクセストークン）
                  </h4>
                  <pre className="bg-blue-50 rounded p-3 text-xs font-mono overflow-x-auto max-h-80 overflow-y-auto">
                    {formatPayload(exchangedPayload)}
                  </pre>
                </div>
              </div>
            </div>

            {/* act Claim Visualization */}
            {"act" in exchangedPayload && exchangedPayload.act != null && (
              <div>
                <h3 className="text-sm font-semibold text-gray-600 mb-1">
                  act クレーム（委任チェーン）
                </h3>
                <pre className="bg-green-50 rounded p-3 text-xs font-mono overflow-x-auto">
                  {JSON.stringify(exchangedPayload.act, null, 2)}
                </pre>
                <p className="text-xs text-gray-500 mt-1">
                  RFC 8693 Section 4.1: act クレームはトークンの委任チェーンを表現します。
                </p>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
