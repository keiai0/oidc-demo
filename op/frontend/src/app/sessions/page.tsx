"use client";

import { useEffect, useState } from "react";
import { Alert } from "@/components/ui/alert";

const API_URL = process.env.NEXT_PUBLIC_OP_BACKEND_BASE_URL || "http://localhost:8080";

type SessionItem = {
  id: string;
  ip_address: string;
  user_agent: string;
  created_at: string;
  expires_at: string;
  is_current: boolean;
};

export default function SessionsPage() {
  const [sessions, setSessions] = useState<SessionItem[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [revoking, setRevoking] = useState<string | null>(null);

  async function loadSessions() {
    try {
      const res = await fetch(`${API_URL}/internal/sessions`, {
        credentials: "include",
      });
      if (!res.ok) {
        setError("セッション一覧の取得に失敗しました");
        return;
      }
      const data: SessionItem[] = await res.json();
      setSessions(data);
    } catch {
      setError("サーバーに接続できません");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadSessions();
  }, []);

  async function handleRevoke(id: string) {
    setRevoking(id);
    try {
      const res = await fetch(`${API_URL}/internal/sessions/${id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (!res.ok) {
        const data = await res.json();
        setError(data.error === "not_found" ? "セッションが見つかりません" : "失効に失敗しました");
        return;
      }
      await loadSessions();
    } catch {
      setError("サーバーに接続できません");
    } finally {
      setRevoking(null);
    }
  }

  function formatDate(iso: string) {
    return new Date(iso).toLocaleString("ja-JP");
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-100">
        <p className="text-gray-600">読み込み中...</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-100 p-6">
      <div className="max-w-2xl mx-auto">
        <h1 className="text-2xl font-semibold text-gray-800 mb-6">アクティブセッション</h1>
        {error && <Alert variant="error">{error}</Alert>}
        <div className="space-y-3">
          {sessions.length === 0 && (
            <p className="text-gray-500 text-sm">アクティブなセッションがありません。</p>
          )}
          {sessions.map((s) => (
            <div
              key={s.id}
              className="bg-white rounded-lg shadow-sm p-4 flex items-start justify-between"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1">
                  {s.is_current && (
                    <span className="inline-block bg-blue-100 text-blue-700 text-xs font-medium px-2 py-0.5 rounded">
                      このデバイス
                    </span>
                  )}
                  <span className="text-sm font-medium text-gray-700 truncate">
                    {s.ip_address}
                  </span>
                </div>
                <p className="text-xs text-gray-500 truncate">{s.user_agent}</p>
                <p className="text-xs text-gray-400 mt-1">
                  ログイン日時: {formatDate(s.created_at)}
                </p>
                <p className="text-xs text-gray-400">
                  有効期限: {formatDate(s.expires_at)}
                </p>
              </div>
              {!s.is_current && (
                <button
                  onClick={() => handleRevoke(s.id)}
                  disabled={revoking === s.id}
                  className="ml-4 shrink-0 px-3 py-1.5 text-sm text-red-600 border border-red-300 rounded hover:bg-red-50 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {revoking === s.id ? "処理中..." : "失効させる"}
                </button>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
