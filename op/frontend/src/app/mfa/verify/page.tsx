"use client";

import { useState, useEffect, type FormEvent } from "react";
import { Alert } from "@/components/ui/alert";

const API_URL =
  process.env.NEXT_PUBLIC_OP_BACKEND_BASE_URL || "http://localhost:8080";

export default function MfaVerifyPage() {
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [redirectAfterMfa, setRedirectAfterMfa] = useState("");

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    setRedirectAfterMfa(params.get("redirect_after_mfa") || "");
  }, []);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await fetch(`${API_URL}/internal/mfa/totp/verify`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          code,
          redirect_after_mfa: redirectAfterMfa,
        }),
      });

      if (!res.ok) {
        const data = await res.json();
        if (data.error === "invalid_code") {
          setError("認証コードが正しくありません");
        } else if (data.error === "mfa_not_pending") {
          setError("MFA 検証が必要な状態ではありません");
        } else if (data.error === "unauthorized") {
          setError("セッションが無効です。再度ログインしてください。");
        } else {
          setError("検証に失敗しました");
        }
        return;
      }

      const data = await res.json();
      if (data.redirect_to) {
        window.location.href = `${API_URL}${data.redirect_to}`;
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
        <h1 className="text-2xl font-semibold text-center text-gray-800 mb-2">
          2段階認証
        </h1>
        <p className="text-sm text-gray-500 text-center mb-6">
          認証アプリに表示されている6桁のコードを入力してください
        </p>
        {error && <Alert variant="error">{error}</Alert>}
        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label
              htmlFor="code"
              className="block text-sm font-medium text-gray-600 mb-1"
            >
              認証コード
            </label>
            <input
              id="code"
              type="text"
              inputMode="numeric"
              autoComplete="one-time-code"
              maxLength={6}
              pattern="[0-9]{6}"
              value={code}
              onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
              placeholder="000000"
              required
              autoFocus
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-center text-2xl tracking-widest"
            />
          </div>
          <button
            type="submit"
            disabled={loading || code.length !== 6}
            className="w-full py-3 bg-blue-600 text-white rounded font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed mt-2"
          >
            {loading ? "確認中..." : "確認"}
          </button>
        </form>
      </div>
    </div>
  );
}
