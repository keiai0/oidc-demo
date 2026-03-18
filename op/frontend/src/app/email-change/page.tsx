"use client";

import { useState } from "react";
import { Alert } from "@/components/ui/alert";

const API_URL = process.env.NEXT_PUBLIC_OP_BACKEND_BASE_URL || "http://localhost:8080";

export default function EmailChangePage() {
  const [newEmail, setNewEmail] = useState("");
  const [error, setError] = useState("");
  const [sent, setSent] = useState(false);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await fetch(`${API_URL}/internal/email/change-request`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ new_email: newEmail }),
      });

      if (!res.ok) {
        const data = await res.json();
        if (data.error === "unauthorized" || data.error === "mfa_required") {
          setError("セッションが無効です。再度ログインしてください。");
        } else if (data.error === "invalid_request") {
          setError(data.error_description || "入力内容を確認してください");
        } else {
          setError("メールアドレス変更のリクエストに失敗しました");
        }
        return;
      }

      setSent(true);
    } catch {
      setError("サーバーに接続できません");
    } finally {
      setLoading(false);
    }
  }

  if (sent) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-100">
        <div className="bg-white p-8 rounded-lg shadow-md w-full max-w-sm">
          <h1 className="text-2xl font-semibold text-center text-gray-800 mb-4">
            確認メールを送信しました
          </h1>
          <p className="text-sm text-gray-600 text-center">
            <span className="font-medium">{newEmail}</span> に確認メールを送信しました。
            メール内のリンクをクリックしてメールアドレスの変更を完了してください。
          </p>
          <p className="text-xs text-gray-400 text-center mt-4">
            リンクの有効期限は24時間です。
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100">
      <div className="bg-white p-8 rounded-lg shadow-md w-full max-w-sm">
        <h1 className="text-2xl font-semibold text-center text-gray-800 mb-6">
          メールアドレス変更
        </h1>
        {error && <Alert variant="error">{error}</Alert>}
        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label
              htmlFor="newEmail"
              className="block text-sm font-medium text-gray-600 mb-1"
            >
              新しいメールアドレス
            </label>
            <input
              id="newEmail"
              type="email"
              value={newEmail}
              onChange={(e) => setNewEmail(e.target.value)}
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
            {loading ? "送信中..." : "確認メールを送信"}
          </button>
        </form>
      </div>
    </div>
  );
}
