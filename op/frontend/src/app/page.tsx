"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

const API_URL = process.env.NEXT_PUBLIC_OP_BACKEND_BASE_URL || "http://localhost:8080";

export default function Home() {
  const [checked, setChecked] = useState(false);

  useEffect(() => {
    fetch(`${API_URL}/internal/me`, { credentials: "include" })
      .then((res) => {
        if (res.ok) {
          window.location.href = "/account";
        } else {
          setChecked(true);
        }
      })
      .catch(() => setChecked(true));
  }, []);

  if (!checked) {
    return (
      <main className="min-h-screen flex items-center justify-center bg-gray-50">
        <p className="text-sm text-gray-400">読み込み中...</p>
      </main>
    );
  }

  return (
    <main className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="w-full max-w-sm space-y-6">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-gray-900">OpenID Provider</h1>
          <p className="text-sm text-gray-500 mt-1">OIDC Demo</p>
        </div>

        <div className="bg-white rounded-lg border border-gray-200 p-6 space-y-3">
          <Link
            href="/login"
            className="block w-full py-2 px-4 bg-blue-600 text-white text-sm text-center rounded hover:bg-blue-700"
          >
            ユーザーログイン
          </Link>
          <Link
            href="/management/login"
            className="block w-full py-2 px-4 border border-gray-300 text-sm text-center text-gray-700 rounded hover:bg-gray-50"
          >
            管理画面ログイン
          </Link>
        </div>
      </div>
    </main>
  );
}
