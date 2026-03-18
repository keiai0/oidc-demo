"use client";

import { useState } from "react";
import { Alert } from "@/components/ui/alert";

const API_URL = process.env.NEXT_PUBLIC_OP_BACKEND_BASE_URL || "http://localhost:8080";

export default function MFAManagePage() {
  // TOTP 無効化
  const [totpPassword, setTotpPassword] = useState("");
  const [totpError, setTotpError] = useState("");
  const [totpSuccess, setTotpSuccess] = useState(false);
  const [totpLoading, setTotpLoading] = useState(false);

  // バックアップコード生成
  const [backupCodes, setBackupCodes] = useState<string[] | null>(null);
  const [backupError, setBackupError] = useState("");
  const [backupLoading, setBackupLoading] = useState(false);

  async function handleTOTPDisable(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setTotpError("");
    setTotpLoading(true);

    try {
      const res = await fetch(`${API_URL}/internal/mfa/totp`, {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ password: totpPassword }),
      });

      if (!res.ok) {
        const data = await res.json();
        if (data.error === "invalid_credentials") {
          setTotpError("パスワードが正しくありません");
        } else if (data.error === "not_found") {
          setTotpError("TOTP は設定されていません");
        } else if (data.error === "unauthorized" || data.error === "mfa_required") {
          setTotpError("セッションが無効です。再度ログインしてください。");
        } else {
          setTotpError("TOTP の無効化に失敗しました");
        }
        return;
      }

      setTotpSuccess(true);
      setTotpPassword("");
    } catch {
      setTotpError("サーバーに接続できません");
    } finally {
      setTotpLoading(false);
    }
  }

  async function handleGenerateBackupCodes() {
    setBackupError("");
    setBackupLoading(true);
    setBackupCodes(null);

    try {
      const res = await fetch(`${API_URL}/internal/mfa/backup-codes/generate`, {
        method: "POST",
        credentials: "include",
      });

      if (!res.ok) {
        const data = await res.json();
        if (data.error === "unauthorized" || data.error === "mfa_required") {
          setBackupError("セッションが無効です。再度ログインしてください。");
        } else {
          setBackupError("バックアップコードの生成に失敗しました");
        }
        return;
      }

      const data = await res.json();
      setBackupCodes(data.codes);
    } catch {
      setBackupError("サーバーに接続できません");
    } finally {
      setBackupLoading(false);
    }
  }

  function handleCopyCodes() {
    if (backupCodes) {
      navigator.clipboard.writeText(backupCodes.join("\n"));
    }
  }

  return (
    <div className="min-h-screen bg-gray-100 p-6">
      <div className="max-w-lg mx-auto space-y-6">
        <h1 className="text-2xl font-semibold text-gray-800">MFA 管理</h1>

        {/* TOTP 無効化 */}
        <div className="bg-white p-6 rounded-lg shadow-sm">
          <h2 className="text-lg font-medium text-gray-700 mb-4">TOTP 無効化</h2>
          {totpSuccess ? (
            <p className="text-sm text-green-600">TOTP が正常に無効化されました。</p>
          ) : (
            <>
              {totpError && <Alert variant="error">{totpError}</Alert>}
              <p className="text-sm text-gray-500 mb-4">
                TOTP を無効化するには現在のパスワードを入力してください。
              </p>
              <form onSubmit={handleTOTPDisable}>
                <div className="mb-4">
                  <label
                    htmlFor="totpPassword"
                    className="block text-sm font-medium text-gray-600 mb-1"
                  >
                    パスワード
                  </label>
                  <input
                    id="totpPassword"
                    type="password"
                    value={totpPassword}
                    onChange={(e) => setTotpPassword(e.target.value)}
                    required
                    className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-red-400 focus:border-transparent"
                  />
                </div>
                <button
                  type="submit"
                  disabled={totpLoading}
                  className="w-full py-2 bg-red-600 text-white rounded font-medium hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {totpLoading ? "処理中..." : "TOTP を無効化"}
                </button>
              </form>
            </>
          )}
        </div>

        {/* バックアップコード生成 */}
        <div className="bg-white p-6 rounded-lg shadow-sm">
          <h2 className="text-lg font-medium text-gray-700 mb-4">バックアップコード</h2>
          {backupError && <Alert variant="error">{backupError}</Alert>}
          <p className="text-sm text-gray-500 mb-4">
            MFA デバイスを紛失した場合のリカバリーコードを生成します。
            生成すると既存のバックアップコードは無効になります。
          </p>
          {backupCodes ? (
            <div className="space-y-3">
              <div className="bg-yellow-50 border border-yellow-200 rounded p-3">
                <p className="text-xs text-yellow-700 font-medium mb-2">
                  ⚠ これらのコードは今後表示されません。安全な場所に保存してください。
                </p>
                <pre className="text-sm font-mono text-gray-800 whitespace-pre-wrap">
                  {backupCodes.join("\n")}
                </pre>
              </div>
              <button
                onClick={handleCopyCodes}
                className="w-full py-2 border border-gray-300 text-gray-700 rounded font-medium hover:bg-gray-50"
              >
                コピー
              </button>
            </div>
          ) : (
            <button
              onClick={handleGenerateBackupCodes}
              disabled={backupLoading}
              className="w-full py-2 bg-blue-600 text-white rounded font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {backupLoading ? "生成中..." : "バックアップコードを生成"}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
