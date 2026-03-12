"use client";

import { useState, useEffect, type FormEvent } from "react";
import { Alert } from "@/components/ui/alert";

const API_URL =
  process.env.NEXT_PUBLIC_OP_BACKEND_BASE_URL || "http://localhost:8080";

type SetupData = {
  secret: string;
  qr_code_uri: string;
  qr_code_png: string;
};

export default function MfaSetupPage() {
  const [step, setStep] = useState<"init" | "scan" | "done">("init");
  const [setupData, setSetupData] = useState<SetupData | null>(null);
  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [redirectAfterMfa, setRedirectAfterMfa] = useState("");

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    setRedirectAfterMfa(params.get("redirect_after_mfa") || "");
  }, []);

  async function handleStartSetup() {
    setError("");
    setLoading(true);

    try {
      const res = await fetch(`${API_URL}/internal/mfa/totp/setup`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
      });

      if (!res.ok) {
        const data = await res.json();
        if (data.error === "mfa_already_configured") {
          setError("TOTP は既に設定されています");
        } else if (data.error === "unauthorized") {
          setError("セッションが無効です。再度ログインしてください。");
        } else {
          setError("セットアップの開始に失敗しました");
        }
        return;
      }

      const data: SetupData = await res.json();
      setSetupData(data);
      setStep("scan");
    } catch {
      setError("サーバーに接続できません");
    } finally {
      setLoading(false);
    }
  }

  async function handleVerify(e: FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await fetch(`${API_URL}/internal/mfa/totp/verify-setup`, {
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
        } else {
          setError("検証に失敗しました");
        }
        return;
      }

      const data = await res.json();

      // ログインフローからの遷移の場合、authorize に戻る
      if (data.redirect_to) {
        window.location.href = `${API_URL}${data.redirect_to}`;
        return;
      }

      setStep("done");
    } catch {
      setError("サーバーに接続できません");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100">
      <div className="bg-white p-8 rounded-lg shadow-md w-full max-w-md">
        <h1 className="text-2xl font-semibold text-center text-gray-800 mb-6">
          2段階認証のセットアップ
        </h1>
        {redirectAfterMfa && (
          <div className="mb-4 p-3 bg-blue-50 rounded border border-blue-200">
            <p className="text-xs text-blue-700">
              このテナントでは2段階認証が必須です。続行するにはセットアップを完了してください。
            </p>
          </div>
        )}
        {error && <Alert variant="error">{error}</Alert>}

        {step === "init" && (
          <div>
            <p className="text-sm text-gray-600 mb-4">
              認証アプリ（Google Authenticator, Authy
              等）を使って2段階認証を設定します。
            </p>
            <button
              onClick={handleStartSetup}
              disabled={loading}
              className="w-full py-3 bg-blue-600 text-white rounded font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? "準備中..." : "セットアップを開始"}
            </button>
          </div>
        )}

        {step === "scan" && setupData && (
          <div>
            <p className="text-sm text-gray-600 mb-4">
              認証アプリで以下の QR コードをスキャンしてください。
            </p>
            <div className="flex justify-center mb-4">
              <img
                src={`data:image/png;base64,${setupData.qr_code_png}`}
                alt="TOTP QR Code"
                width={256}
                height={256}
              />
            </div>
            <div className="mb-4 p-3 bg-gray-50 rounded border border-gray-200">
              <p className="text-xs text-gray-500 mb-1">
                QR コードを読み取れない場合は、以下のキーを手動で入力してください:
              </p>
              <p className="font-mono text-sm text-gray-800 break-all select-all">
                {setupData.secret}
              </p>
            </div>
            <form onSubmit={handleVerify}>
              <div className="mb-4">
                <label
                  htmlFor="code"
                  className="block text-sm font-medium text-gray-600 mb-1"
                >
                  認証コードを入力して確認
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
                className="w-full py-3 bg-blue-600 text-white rounded font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loading ? "確認中..." : "有効化"}
              </button>
            </form>
          </div>
        )}

        {step === "done" && (
          <div className="text-center">
            <div className="text-green-500 text-5xl mb-4">&#10003;</div>
            <p className="text-lg font-medium text-gray-800 mb-2">
              2段階認証が有効になりました
            </p>
            <p className="text-sm text-gray-500">
              次回のログインから認証コードの入力が必要になります。
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
