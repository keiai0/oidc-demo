"use client";

import { useState, useEffect } from "react";
import { Alert } from "@/components/ui/alert";

const API_URL = process.env.NEXT_PUBLIC_OP_BACKEND_BASE_URL || "http://localhost:8080";

export default function PasswordResetRequestPage() {
  const [email, setEmail] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);
  const [loading, setLoading] = useState(false);
  const [tenantCode, setTenantCode] = useState("");

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    setTenantCode(params.get("tenant_code") || "demo");
  }, []);

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await fetch(`${API_URL}/internal/password/reset-request`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ tenant_code: tenantCode, email }),
      });

      if (!res.ok) {
        setError("リクエストに失敗しました");
        return;
      }

      setSuccess(true);
    } catch {
      setError("サーバーに接続できません");
    } finally {
      setLoading(false);
    }
  }

  if (success) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-100">
        <div className="bg-white p-8 rounded-lg shadow-md w-full max-w-sm">
          <h1 className="text-2xl font-semibold text-center text-gray-800 mb-4">
            メール送信完了
          </h1>
          <p className="text-sm text-gray-600 text-center mb-4">
            アカウントが存在する場合、パスワードリセット用のメールを送信しました。メールの指示に従ってパスワードをリセットしてください。
          </p>
          <a
            href={`/login?tenant_code=${tenantCode}`}
            className="block w-full py-3 bg-blue-600 text-white rounded font-medium hover:bg-blue-700 text-center"
          >
            ログインに戻る
          </a>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100">
      <div className="bg-white p-8 rounded-lg shadow-md w-full max-w-sm">
        <h1 className="text-2xl font-semibold text-center text-gray-800 mb-6">
          パスワードリセット
        </h1>
        <p className="text-sm text-gray-600 mb-4">
          登録済みのメールアドレスを入力してください。リセット用のリンクを送信します。
        </p>
        <div className="mb-4 p-3 bg-gray-50 rounded border border-gray-200 text-xs text-gray-500">
          <p className="font-medium text-gray-600 mb-1">テスト用アカウント</p>
          <p>メールアドレス: <code className="bg-gray-100 px-1 rounded">testuser@example.com</code></p>
        </div>
        {error && <Alert variant="error">{error}</Alert>}
        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label
              htmlFor="email"
              className="block text-sm font-medium text-gray-600 mb-1"
            >
              メールアドレス
            </label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              autoFocus
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="w-full py-3 bg-blue-600 text-white rounded font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed mt-2"
          >
            {loading ? "送信中..." : "リセットリンクを送信"}
          </button>
        </form>
        <div className="mt-4 text-center">
          <a
            href={`/login?tenant_code=${tenantCode}`}
            className="text-sm text-blue-600 hover:underline"
          >
            ログインに戻る
          </a>
        </div>
      </div>
    </div>
  );
}
