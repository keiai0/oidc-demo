"use client";

import { useState, useEffect, useCallback } from "react";
import Link from "next/link";

// --- シナリオ定義 ---

type Scenario = {
  id: string;
  label: string;
  description: string;
  authorizationDetails: Record<string, unknown>[];
  scope: string;
};

const SCENARIOS: Scenario[] = [
  {
    id: "payment",
    label: "送金許可 (Payment Initiation)",
    description:
      "RFC 9396 Section 2 に基づく送金認可リクエスト。特定口座からの送金操作を認可します。",
    authorizationDetails: [
      {
        type: "payment_initiation",
        actions: ["initiate"],
        identifier: "account-123",
        amount: { value: "100", currency: "JPY" },
        recipient: "田中太郎",
      },
    ],
    scope: "openid",
  },
  {
    id: "account",
    label: "口座情報照会 (Account Information)",
    description:
      "口座情報の読み取りアクセスを要求するリクエスト。期間指定付きでリソースの場所も指定します。",
    authorizationDetails: [
      {
        type: "account_information",
        actions: ["read"],
        locations: ["https://example.com/accounts"],
        from_date: "2025-01-01",
        to_date: "2025-12-31",
      },
    ],
    scope: "openid",
  },
];

// --- 型定義 ---

type OPConfig = {
  authorization_endpoint: string;
  client_id: string;
  redirect_uri: string;
};

type TokenResponse = {
  access_token?: string;
  id_token?: string;
  token_type?: string;
  expires_in?: number;
  authorization_details?: Record<string, unknown>[];
  error?: string;
  error_description?: string;
};

export default function RARDemoPage() {
  const [selectedScenario, setSelectedScenario] = useState<string>("payment");
  const [opConfig, setOpConfig] = useState<OPConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [tokenResponse, setTokenResponse] = useState<TokenResponse | null>(
    null,
  );
  const [authCode, setAuthCode] = useState<string | null>(null);

  const scenario = SCENARIOS.find((s) => s.id === selectedScenario)!;

  // OP の設定を取得
  useEffect(() => {
    fetch("/api/demo/rar", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: "get-config" }),
    })
      .then((res) => res.json())
      .then(setOpConfig)
      .catch(() => setError("OP 設定の取得に失敗しました"));
  }, []);

  // コールバック: URL に code パラメータがあれば取得
  const handleCodeExchange = useCallback(
    async (code: string) => {
      if (!opConfig) return;

      setLoading(true);
      setError("");
      try {
        const res = await fetch("/api/demo/rar", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            action: "exchange",
            code,
            redirect_uri: opConfig.redirect_uri,
          }),
        });
        const data = await res.json();
        if (!res.ok) {
          throw new Error(
            data.error_description || data.error || `HTTP ${res.status}`,
          );
        }
        setTokenResponse(data);
      } catch (e) {
        const msg =
          e instanceof Error ? e.message : "トークン交換に失敗しました";
        setError(msg);
      } finally {
        setLoading(false);
      }
    },
    [opConfig],
  );

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const code = params.get("code");
    const errorParam = params.get("error");

    if (errorParam) {
      const desc =
        params.get("error_description") ?? "認可サーバーからエラーが返されました";
      setError(`${errorParam}: ${desc}`);
      // URL をクリーンアップ
      window.history.replaceState({}, "", window.location.pathname);
      return;
    }

    if (code) {
      setAuthCode(code);
      // URL をクリーンアップ（コードが URL に残り続けるのを防ぐ）
      window.history.replaceState({}, "", window.location.pathname);
    }
  }, []);

  // コード取得後にトークン交換を実行
  useEffect(() => {
    if (authCode && opConfig && !tokenResponse) {
      handleCodeExchange(authCode);
    }
  }, [authCode, opConfig, tokenResponse, handleCodeExchange]);

  // 認可リクエスト送信
  const handleAuthorize = () => {
    if (!opConfig) {
      setError("OP 設定がまだ読み込まれていません");
      return;
    }

    const state = crypto.randomUUID();

    const params = new URLSearchParams({
      response_type: "code",
      client_id: opConfig.client_id,
      redirect_uri: opConfig.redirect_uri,
      scope: scenario.scope,
      state,
      authorization_details: JSON.stringify(scenario.authorizationDetails),
    });

    const authorizeUrl = `${opConfig.authorization_endpoint}?${params.toString()}`;
    window.location.href = authorizeUrl;
  };

  // 認可 URL のプレビュー生成
  const buildPreviewUrl = () => {
    if (!opConfig) return "(読み込み中...)";

    const params = new URLSearchParams({
      response_type: "code",
      client_id: opConfig.client_id,
      redirect_uri: opConfig.redirect_uri,
      scope: scenario.scope,
      state: "<random>",
      authorization_details: JSON.stringify(scenario.authorizationDetails),
    });

    return `${opConfig.authorization_endpoint}?${params.toString()}`;
  };

  // リセット
  const handleReset = () => {
    setTokenResponse(null);
    setAuthCode(null);
    setError("");
  };

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-3xl mx-auto px-4 space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold">
            Rich Authorization Requests デモ
          </h1>
          <Link href="/" className="text-blue-600 hover:underline text-sm">
            ← ホームに戻る
          </Link>
        </div>

        <p className="text-sm text-gray-600">
          RFC 9396 に基づく Rich Authorization Requests (RAR)
          のデモページです。従来の scope ベースではなく、
          <code className="bg-gray-100 px-1 rounded">
            authorization_details
          </code>{" "}
          パラメータで構造化された認可要求を送信します。
        </p>

        {error && (
          <div className="p-3 bg-red-50 border border-red-200 rounded text-sm text-red-700">
            {error}
          </div>
        )}

        {/* シナリオ選択 */}
        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <h2 className="text-lg font-semibold mb-4">1. シナリオ選択</h2>
          <div className="flex gap-3 mb-4">
            {SCENARIOS.map((s) => (
              <button
                key={s.id}
                onClick={() => {
                  setSelectedScenario(s.id);
                  handleReset();
                }}
                className={`px-4 py-2 text-sm rounded border transition-colors ${
                  selectedScenario === s.id
                    ? "bg-blue-600 text-white border-blue-600"
                    : "bg-white text-gray-700 border-gray-300 hover:bg-gray-50"
                }`}
              >
                {s.label}
              </button>
            ))}
          </div>
          <p className="text-sm text-gray-600">{scenario.description}</p>
        </div>

        {/* authorization_details 表示 */}
        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <h2 className="text-lg font-semibold mb-4">
            2. authorization_details パラメータ
          </h2>
          <p className="text-sm text-gray-600 mb-3">
            認可リクエストに含まれる構造化された認可要求の JSON です。
          </p>
          <div className="p-3 bg-gray-900 rounded border">
            <pre className="text-xs text-green-400 overflow-auto whitespace-pre-wrap">
              {JSON.stringify(scenario.authorizationDetails, null, 2)}
            </pre>
          </div>
        </div>

        {/* scope との比較 */}
        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <h2 className="text-lg font-semibold mb-4">
            3. scope ベース vs RAR の比較
          </h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-2">
                従来の scope ベース
              </h3>
              <div className="p-3 bg-gray-50 rounded border text-xs font-mono">
                scope=openid{" "}
                {selectedScenario === "payment"
                  ? "payment:initiate"
                  : "account:read"}
              </div>
              <p className="text-xs text-gray-500 mt-2">
                scope
                はアクセス範囲の粗い指定のみ。具体的な操作対象やパラメータを含められない。
              </p>
            </div>
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-2">
                RAR (authorization_details)
              </h3>
              <div className="p-3 bg-blue-50 rounded border border-blue-200 text-xs font-mono">
                authorization_details=
                {JSON.stringify(scenario.authorizationDetails).slice(0, 60)}
                ...
              </div>
              <p className="text-xs text-gray-500 mt-2">
                操作対象・金額・期間など、具体的な認可パラメータを構造化して送信できる。
              </p>
            </div>
          </div>
        </div>

        {/* 認可リクエスト送信 */}
        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <h2 className="text-lg font-semibold mb-4">4. 認可リクエスト送信</h2>
          {!tokenResponse ? (
            <>
              <p className="text-sm text-gray-600 mb-3">
                以下の URL に認可リクエストを送信します（ブラウザリダイレクト）。
              </p>
              <div className="p-3 bg-gray-50 rounded border mb-4">
                <p className="text-xs font-mono text-gray-600 break-all">
                  {buildPreviewUrl()}
                </p>
              </div>
              <button
                onClick={handleAuthorize}
                disabled={loading || !opConfig}
                className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700 disabled:opacity-50"
              >
                {loading ? "処理中..." : "認可リクエスト送信"}
              </button>
            </>
          ) : (
            <div className="space-y-4">
              <div className="p-3 bg-green-50 border border-green-200 rounded text-sm text-green-700">
                トークン交換が完了しました
              </div>
              <button
                onClick={handleReset}
                className="px-4 py-2 bg-gray-600 text-white text-sm rounded hover:bg-gray-700"
              >
                リセット
              </button>
            </div>
          )}
        </div>

        {/* トークンレスポンス */}
        {tokenResponse && (
          <div className="bg-white p-6 rounded-lg shadow-sm border">
            <h2 className="text-lg font-semibold mb-4">
              5. トークンレスポンス
            </h2>

            {/* authorization_details をハイライト表示 */}
            {tokenResponse.authorization_details && (
              <div className="mb-4">
                <h3 className="text-sm font-medium text-gray-700 mb-2">
                  authorization_details（トークンレスポンスに含まれる認可詳細）
                </h3>
                <div className="p-3 bg-blue-50 rounded border border-blue-200">
                  <pre className="text-xs text-blue-800 overflow-auto whitespace-pre-wrap">
                    {JSON.stringify(
                      tokenResponse.authorization_details,
                      null,
                      2,
                    )}
                  </pre>
                </div>
                <p className="text-xs text-gray-500 mt-2">
                  RFC 9396 Section 7: 認可サーバーは許可された authorization_details
                  をトークンレスポンスに含めて返します。
                  リソースサーバーはこの情報を使ってアクセス制御を行います。
                </p>
              </div>
            )}

            {!tokenResponse.authorization_details && (
              <div className="mb-4 p-3 bg-yellow-50 border border-yellow-200 rounded">
                <p className="text-sm text-yellow-700">
                  トークンレスポンスに authorization_details が含まれていません。
                  OP が RAR をサポートしているか確認してください。
                </p>
              </div>
            )}

            {/* フルレスポンス */}
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-2">
                完全なトークンレスポンス
              </h3>
              <div className="p-3 bg-gray-900 rounded border">
                <pre className="text-xs text-green-400 overflow-auto whitespace-pre-wrap">
                  {JSON.stringify(tokenResponse, null, 2)}
                </pre>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
