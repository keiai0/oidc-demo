"use client";

import { useState, useEffect, type FormEvent } from "react";
import { Alert } from "@/components/ui/alert";

const API_URL = process.env.NEXT_PUBLIC_OP_BACKEND_BASE_URL || "http://localhost:8080";

export default function LoginPage() {
  const [loginId, setLoginId] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [tenantCode, setTenantCode] = useState("");
  const [redirectAfterLogin, setRedirectAfterLogin] = useState("");

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    setTenantCode(params.get("tenant_code") || "demo");
    setRedirectAfterLogin(params.get("redirect_after_login") || "");
  }, []);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await fetch(`${API_URL}/internal/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          tenant_code: tenantCode,
          login_id: loginId,
          password: password,
        }),
      });

      if (!res.ok) {
        const data = await res.json();
        setError(
          data.error === "invalid_credentials"
            ? "ログインIDまたはパスワードが正しくありません"
            : "ログインに失敗しました",
        );
        return;
      }

      if (redirectAfterLogin) {
        // redirect_after_login は OP Backend の相対パス（例: /demo/authorize?...）
        // OP Frontend からのリダイレクトなので OP Backend の絶対 URL に変換する
        window.location.href = `${API_URL}${redirectAfterLogin}`;
      }
    } catch {
      setError("サーバーに接続できません");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100">
      <div className="bg-white p-8 rounded-lg shadow-md w-full max-w-sm">
        <h1 className="text-2xl font-semibold text-center text-gray-800 mb-6">
          ログイン
        </h1>
        {error && <Alert variant="error">{error}</Alert>}
        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label
              htmlFor="loginId"
              className="block text-sm font-medium text-gray-600 mb-1"
            >
              ログインID
            </label>
            <input
              id="loginId"
              type="text"
              value={loginId}
              onChange={(e) => setLoginId(e.target.value)}
              required
              autoFocus
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
          <div className="mb-4">
            <label
              htmlFor="password"
              className="block text-sm font-medium text-gray-600 mb-1"
            >
              パスワード
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="w-full py-3 bg-blue-600 text-white rounded font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed mt-2"
          >
            {loading ? "ログイン中..." : "ログイン"}
          </button>
        </form>
      </div>
    </div>
  );
}
