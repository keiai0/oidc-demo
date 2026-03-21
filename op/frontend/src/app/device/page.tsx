"use client";

import { useState, useEffect } from "react";
import { Alert } from "@/components/ui/alert";

const API_URL =
  process.env.NEXT_PUBLIC_OP_BACKEND_BASE_URL || "http://localhost:8080";

type DeviceInfo = {
  user_code: string;
  scope: string;
  client_name: string;
  expires_at: string;
};

export default function DeviceVerifyPage() {
  const [userCode, setUserCode] = useState("");
  const [deviceInfo, setDeviceInfo] = useState<DeviceInfo | null>(null);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const code = params.get("user_code") || "";
    if (code) {
      setUserCode(code);
    }
  }, []);

  // user_code がURLパラメータで渡された場合、自動的にデバイス情報を取得
  useEffect(() => {
    if (userCode && userCode.length >= 9) {
      handleLookup();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function handleLookup() {
    setError("");
    setDeviceInfo(null);
    setLoading(true);
    try {
      const res = await fetch(
        `${API_URL}/internal/device/verify?user_code=${encodeURIComponent(userCode)}`,
        { credentials: "include" }
      );

      if (res.status === 401) {
        // 未認証: ログインページへリダイレクト
        window.location.href = `/login?redirect_after_login=${encodeURIComponent(window.location.href)}`;
        return;
      }

      const data = await res.json();
      if (!res.ok) {
        if (data.error === "expired") {
          setError("このコードは有効期限切れです。デバイスで再度コードを取得してください。");
        } else if (data.error === "invalid_user_code") {
          setError("無効なコードです。入力内容を確認してください。");
        } else if (data.error === "already_processed") {
          setError("このリクエストは既に処理済みです。");
        } else {
          setError(data.error || "エラーが発生しました");
        }
        return;
      }

      setDeviceInfo(data);
    } catch {
      setError("サーバーに接続できませんでした");
    } finally {
      setLoading(false);
    }
  }

  async function handleDecision(approved: boolean) {
    setError("");
    setLoading(true);
    try {
      const endpoint = approved
        ? "/internal/device/approve"
        : "/internal/device/deny";

      const res = await fetch(`${API_URL}${endpoint}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ user_code: userCode }),
      });

      if (res.status === 401) {
        window.location.href = `/login?redirect_after_login=${encodeURIComponent(window.location.href)}`;
        return;
      }

      const data = await res.json();
      if (!res.ok) {
        setError(data.error || "エラーが発生しました");
        return;
      }

      setDeviceInfo(null);
      if (approved) {
        setSuccess("デバイスを承認しました。デバイス側でログインが完了します。");
      } else {
        setSuccess("デバイスのリクエストを拒否しました。");
      }
    } catch {
      setError("サーバーに接続できませんでした");
    } finally {
      setLoading(false);
    }
  }

  // user_code の入力フォーマット: 入力中に自動ハイフン挿入
  function handleCodeChange(value: string) {
    const cleaned = value.replace(/[^A-Za-z]/g, "").toUpperCase();
    if (cleaned.length <= 4) {
      setUserCode(cleaned);
    } else {
      setUserCode(cleaned.slice(0, 4) + "-" + cleaned.slice(4, 8));
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100">
      <div className="bg-white p-8 rounded-lg shadow-md w-full max-w-md">
        <h1 className="text-2xl font-semibold text-center text-gray-800 mb-6">
          デバイス認証
        </h1>
        <p className="text-sm text-gray-600 text-center mb-6">
          デバイスに表示されたコードを入力してください
        </p>

        {error && (
          <Alert variant="error" className="mb-4">
            {error}
          </Alert>
        )}
        {success && (
          <Alert variant="success" className="mb-4">
            {success}
          </Alert>
        )}

        {!deviceInfo && !success && (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleLookup();
            }}
          >
            <div className="mb-4">
              <label
                htmlFor="user_code"
                className="block text-sm font-medium text-gray-600 mb-1"
              >
                コード
              </label>
              <input
                id="user_code"
                type="text"
                value={userCode}
                onChange={(e) => handleCodeChange(e.target.value)}
                placeholder="XXXX-XXXX"
                maxLength={9}
                autoFocus
                className="w-full px-4 py-3 text-center text-2xl font-mono tracking-widest border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            <button
              type="submit"
              disabled={loading || userCode.length < 9}
              className="w-full bg-blue-600 text-white py-2 px-4 rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {loading ? "確認中..." : "確認"}
            </button>
          </form>
        )}

        {deviceInfo && (
          <div>
            <div className="bg-gray-50 rounded-lg p-4 mb-6">
              <div className="mb-3">
                <span className="text-sm text-gray-500">アプリケーション</span>
                <p className="font-medium text-gray-800">
                  {deviceInfo.client_name}
                </p>
              </div>
              {deviceInfo.scope && (
                <div className="mb-3">
                  <span className="text-sm text-gray-500">
                    リクエストされた権限
                  </span>
                  <div className="flex flex-wrap gap-1 mt-1">
                    {deviceInfo.scope.split(" ").map((scope) => (
                      <span
                        key={scope}
                        className="inline-block bg-blue-100 text-blue-800 text-xs px-2 py-1 rounded"
                      >
                        {scope}
                      </span>
                    ))}
                  </div>
                </div>
              )}
              <div>
                <span className="text-sm text-gray-500">コード</span>
                <p className="font-mono text-lg text-gray-800">
                  {deviceInfo.user_code}
                </p>
              </div>
            </div>

            <div className="flex gap-3">
              <button
                onClick={() => handleDecision(false)}
                disabled={loading}
                className="flex-1 border border-gray-300 text-gray-700 py-2 px-4 rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                拒否
              </button>
              <button
                onClick={() => handleDecision(true)}
                disabled={loading}
                className="flex-1 bg-blue-600 text-white py-2 px-4 rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {loading ? "処理中..." : "承認"}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
