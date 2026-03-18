"use client";

import { useState } from "react";
import Link from "next/link";

const API_URL = process.env.NEXT_PUBLIC_OP_BACKEND_BASE_URL || "http://localhost:8080";

export default function AccountPage() {
  const [logoutLoading, setLogoutLoading] = useState(false);

  async function handleLogout() {
    setLogoutLoading(true);
    try {
      await fetch(`${API_URL}/internal/logout`, {
        method: "POST",
        credentials: "include",
      });
    } finally {
      window.location.href = "/";
    }
  }

  return (
    <div className="min-h-screen bg-gray-100 p-6">
      <div className="max-w-sm mx-auto space-y-4">
        <h1 className="text-2xl font-semibold text-gray-800">アカウント設定</h1>

        <div className="bg-white rounded-lg shadow-sm divide-y divide-gray-100">
          <Link
            href="/email-change"
            className="flex items-center justify-between px-4 py-3 hover:bg-gray-50"
          >
            <span className="text-sm text-gray-700">メールアドレス変更</span>
            <span className="text-gray-400">›</span>
          </Link>
          <Link
            href="/password-change"
            className="flex items-center justify-between px-4 py-3 hover:bg-gray-50"
          >
            <span className="text-sm text-gray-700">パスワード変更</span>
            <span className="text-gray-400">›</span>
          </Link>
          <Link
            href="/mfa/manage"
            className="flex items-center justify-between px-4 py-3 hover:bg-gray-50"
          >
            <span className="text-sm text-gray-700">MFA 管理</span>
            <span className="text-gray-400">›</span>
          </Link>
          <Link
            href="/sessions"
            className="flex items-center justify-between px-4 py-3 hover:bg-gray-50"
          >
            <span className="text-sm text-gray-700">セッション管理</span>
            <span className="text-gray-400">›</span>
          </Link>
        </div>

        <button
          onClick={handleLogout}
          disabled={logoutLoading}
          className="w-full py-2 text-sm text-red-600 border border-red-300 rounded hover:bg-red-50 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {logoutLoading ? "ログアウト中..." : "ログアウト"}
        </button>
      </div>
    </div>
  );
}
