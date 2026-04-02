"use client";

import { useState } from "react";
import Link from "next/link";

type RegistrationResult = {
  client_id: string;
  client_secret?: string;
  registration_access_token: string;
  registration_client_uri: string;
  redirect_uris: string[];
  grant_types: string[];
  response_types: string[];
  token_endpoint_auth_method: string;
  client_name?: string;
  client_id_issued_at: number;
  client_secret_expires_at?: number;
};

type ClientConfig = {
  client_id: string;
  registration_access_token?: string;
  registration_client_uri: string;
  redirect_uris: string[];
  grant_types: string[];
  response_types: string[];
  token_endpoint_auth_method: string;
  client_name?: string;
  client_id_issued_at: number;
};

export default function DynamicRegistrationPage() {
  // Registration
  const [iat, setIat] = useState("");
  const [clientName, setClientName] = useState("Demo Dynamic Client");
  const [redirectUri, setRedirectUri] = useState("http://localhost:3001/api/auth/callback");

  // Result
  const [regResult, setRegResult] = useState<RegistrationResult | null>(null);
  const [clientConfig, setClientConfig] = useState<ClientConfig | null>(null);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [loading, setLoading] = useState(false);

  // For subsequent operations
  const [regAccessToken, setRegAccessToken] = useState("");
  const [registeredClientId, setRegisteredClientId] = useState("");

  const callApi = async (body: Record<string, unknown>) => {
    setLoading(true);
    setError("");
    setSuccess("");
    try {
      const res = await fetch("/api/demo/dynamic-registration", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error_description || data.error || `HTTP ${res.status}`);
      }
      return data;
    } catch (e) {
      const msg = e instanceof Error ? e.message : "リクエストに失敗しました";
      setError(msg);
      return null;
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async () => {
    if (!iat) {
      setError("Initial Access Token を入力してください");
      return;
    }

    const data = await callApi({
      action: "register",
      initialAccessToken: iat,
      metadata: {
        client_name: clientName,
        redirect_uris: [redirectUri],
        grant_types: ["authorization_code", "refresh_token"],
        response_types: ["code"],
        token_endpoint_auth_method: "client_secret_basic",
      },
    });

    if (data) {
      setRegResult(data);
      setRegisteredClientId(data.client_id);
      setRegAccessToken(data.registration_access_token);
      setSuccess("クライアントを動的登録しました");
    }
  };

  const handleGetConfig = async () => {
    if (!registeredClientId || !regAccessToken) {
      setError("先にクライアントを登録してください");
      return;
    }

    const data = await callApi({
      action: "get",
      clientId: registeredClientId,
      registrationAccessToken: regAccessToken,
    });

    if (data) {
      setClientConfig(data);
      setSuccess("クライアント情報を取得しました");
    }
  };

  const handleUpdate = async () => {
    if (!registeredClientId || !regAccessToken) {
      setError("先にクライアントを登録してください");
      return;
    }

    const data = await callApi({
      action: "update",
      clientId: registeredClientId,
      registrationAccessToken: regAccessToken,
      metadata: {
        client_name: clientName + " (Updated)",
        redirect_uris: [redirectUri],
        grant_types: ["authorization_code", "refresh_token"],
        response_types: ["code"],
        token_endpoint_auth_method: "client_secret_basic",
      },
    });

    if (data) {
      setClientConfig(data);
      // Registration Access Token がローテーションされるので更新
      if (data.registration_access_token) {
        setRegAccessToken(data.registration_access_token);
      }
      setSuccess("クライアント情報を更新しました（Registration Access Token がローテーションされました）");
    }
  };

  const handleDelete = async () => {
    if (!registeredClientId || !regAccessToken) {
      setError("先にクライアントを登録してください");
      return;
    }
    if (!confirm("登録したクライアントを削除しますか？")) return;

    const data = await callApi({
      action: "delete",
      clientId: registeredClientId,
      registrationAccessToken: regAccessToken,
    });

    if (data) {
      setSuccess("クライアントを削除しました");
      setRegResult(null);
      setClientConfig(null);
      setRegisteredClientId("");
      setRegAccessToken("");
    }
  };

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-3xl mx-auto px-4 space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold">Dynamic Client Registration デモ</h1>
          <Link href="/" className="text-blue-600 hover:underline text-sm">
            ← ホームに戻る
          </Link>
        </div>

        <p className="text-sm text-gray-600">
          RFC 7591 / 7592 に基づく Dynamic Client Registration のデモページです。
          管理画面で発行した Initial Access Token を使ってクライアントを動的に登録・管理できます。
        </p>

        {error && (
          <div className="p-3 bg-red-50 border border-red-200 rounded text-sm text-red-700">
            {error}
          </div>
        )}
        {success && (
          <div className="p-3 bg-green-50 border border-green-200 rounded text-sm text-green-700">
            {success}
          </div>
        )}

        {/* Step 1: 登録 */}
        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <h2 className="text-lg font-semibold mb-4">1. クライアント登録 (RFC 7591)</h2>
          <div className="space-y-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Initial Access Token
              </label>
              <input
                type="text"
                value={iat}
                onChange={(e) => setIat(e.target.value)}
                placeholder="管理画面で発行した IAT を貼り付け"
                className="w-full border border-gray-300 rounded px-3 py-2 text-sm font-mono"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                クライアント名
              </label>
              <input
                type="text"
                value={clientName}
                onChange={(e) => setClientName(e.target.value)}
                className="w-full border border-gray-300 rounded px-3 py-2 text-sm"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                リダイレクト URI
              </label>
              <input
                type="text"
                value={redirectUri}
                onChange={(e) => setRedirectUri(e.target.value)}
                className="w-full border border-gray-300 rounded px-3 py-2 text-sm font-mono"
              />
            </div>
            <button
              onClick={handleRegister}
              disabled={loading}
              className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700 disabled:opacity-50"
            >
              {loading ? "登録中..." : "クライアントを登録"}
            </button>
          </div>
        </div>

        {/* 登録結果 */}
        {regResult && (
          <div className="bg-white p-6 rounded-lg shadow-sm border">
            <h2 className="text-lg font-semibold mb-4">登録結果</h2>
            <div className="space-y-2 text-sm">
              <div>
                <span className="font-medium text-gray-600">client_id: </span>
                <code className="bg-gray-100 px-1 rounded">{regResult.client_id}</code>
              </div>
              {regResult.client_secret && (
                <div>
                  <span className="font-medium text-gray-600">client_secret: </span>
                  <code className="bg-yellow-50 border border-yellow-200 px-1 rounded text-xs break-all">
                    {regResult.client_secret}
                  </code>
                  <span className="text-xs text-red-500 ml-2">（一度だけ表示）</span>
                </div>
              )}
              <div>
                <span className="font-medium text-gray-600">registration_access_token: </span>
                <code className="bg-gray-100 px-1 rounded text-xs break-all">
                  {regResult.registration_access_token}
                </code>
              </div>
              <div>
                <span className="font-medium text-gray-600">registration_client_uri: </span>
                <code className="bg-gray-100 px-1 rounded text-xs">{regResult.registration_client_uri}</code>
              </div>
            </div>
          </div>
        )}

        {/* Step 2: 設定操作 (RFC 7592) */}
        {registeredClientId && (
          <div className="bg-white p-6 rounded-lg shadow-sm border">
            <h2 className="text-lg font-semibold mb-4">
              2. クライアント設定管理 (RFC 7592)
            </h2>
            <div className="flex gap-3">
              <button
                onClick={handleGetConfig}
                disabled={loading}
                className="px-4 py-2 bg-gray-600 text-white text-sm rounded hover:bg-gray-700 disabled:opacity-50"
              >
                情報取得 (GET)
              </button>
              <button
                onClick={handleUpdate}
                disabled={loading}
                className="px-4 py-2 bg-yellow-600 text-white text-sm rounded hover:bg-yellow-700 disabled:opacity-50"
              >
                更新 (PUT)
              </button>
              <button
                onClick={handleDelete}
                disabled={loading}
                className="px-4 py-2 bg-red-600 text-white text-sm rounded hover:bg-red-700 disabled:opacity-50"
              >
                削除 (DELETE)
              </button>
            </div>

            {clientConfig && (
              <div className="mt-4 p-3 bg-gray-50 rounded border">
                <h3 className="text-sm font-medium mb-2">クライアント設定</h3>
                <pre className="text-xs overflow-auto">
                  {JSON.stringify(clientConfig, null, 2)}
                </pre>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
