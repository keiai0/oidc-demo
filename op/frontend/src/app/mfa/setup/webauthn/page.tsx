"use client";

import { useState, useEffect } from "react";
import { Alert } from "@/components/ui/alert";
import {
  prepareCreationOptions,
  registrationCredentialToJSON,
} from "@/lib/webauthn";

const API_URL =
  process.env.NEXT_PUBLIC_OP_BACKEND_BASE_URL || "http://localhost:8080";

export default function WebAuthnSetupPage() {
  const [step, setStep] = useState<"init" | "done">("init");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [redirectAfterMfa, setRedirectAfterMfa] = useState("");
  const [redirectAfterLogin, setRedirectAfterLogin] = useState("");
  const [isSuggest, setIsSuggest] = useState(false);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    setRedirectAfterMfa(params.get("redirect_after_mfa") || "");
    setRedirectAfterLogin(params.get("redirect_after_login") || "");
    setIsSuggest(params.get("suggest") === "1");
  }, []);

  function handleSkip() {
    if (redirectAfterLogin) {
      window.location.href = `${API_URL}${redirectAfterLogin}`;
    } else {
      window.location.href = "/";
    }
  }

  async function handleRegister() {
    setError("");
    setLoading(true);

    try {
      // 1. Begin: チャレンジ取得
      const beginRes = await fetch(
        `${API_URL}/internal/mfa/webauthn/register/begin`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
        }
      );

      if (!beginRes.ok) {
        const data = await beginRes.json();
        if (data.error === "unauthorized") {
          setError("セッションが無効です。再度ログインしてください。");
        } else {
          setError("パスキー登録の開始に失敗しました");
        }
        return;
      }

      const serverOptions = await beginRes.json();

      // 2. ブラウザの WebAuthn API を呼び出し
      const creationOptions = prepareCreationOptions(serverOptions);
      const credential = (await navigator.credentials.create(
        creationOptions
      )) as PublicKeyCredential | null;

      if (!credential) {
        setError("パスキーの作成がキャンセルされました");
        return;
      }

      // 3. Complete: Attestation をサーバーに送信
      const credJSON = registrationCredentialToJSON(credential);
      const completeRes = await fetch(
        `${API_URL}/internal/mfa/webauthn/register/complete`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify(credJSON),
        }
      );

      if (!completeRes.ok) {
        const data = await completeRes.json();
        setError(data.error_description || "パスキーの登録に失敗しました");
        return;
      }

      const data = await completeRes.json();

      // MFA 強制セットアップからの遷移
      if (data.redirect_to) {
        window.location.href = `${API_URL}${data.redirect_to}`;
        return;
      }

      if (redirectAfterMfa) {
        window.location.href = `${API_URL}${redirectAfterMfa}`;
        return;
      }

      // 提案モードで登録完了 → 元のリダイレクト先へ
      if (isSuggest && redirectAfterLogin) {
        setStep("done");
        // 少し完了メッセージを見せてからリダイレクト
        setTimeout(() => {
          window.location.href = `${API_URL}${redirectAfterLogin}`;
        }, 1500);
        return;
      }

      setStep("done");
    } catch (e) {
      if (e instanceof DOMException && e.name === "NotAllowedError") {
        setError("パスキーの作成がキャンセルされました");
      } else {
        setError("エラーが発生しました");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100">
      <div className="bg-white p-8 rounded-lg shadow-md w-full max-w-md">
        {isSuggest ? (
          <h1 className="text-2xl font-semibold text-center text-gray-800 mb-2">
            パスキーを設定しませんか？
          </h1>
        ) : (
          <h1 className="text-2xl font-semibold text-center text-gray-800 mb-6">
            パスキーの登録
          </h1>
        )}

        {isSuggest && step === "init" && (
          <p className="text-sm text-gray-500 text-center mb-6">
            指紋や顔認証でログインできるようになります
          </p>
        )}

        {!isSuggest && redirectAfterMfa && (
          <div className="mb-4 p-3 bg-blue-50 rounded border border-blue-200">
            <p className="text-xs text-blue-700">
              このテナントでは2段階認証が必須です。続行するにはパスキーを登録してください。
            </p>
          </div>
        )}

        {error && <Alert variant="error">{error}</Alert>}

        {step === "init" && (
          <div>
            {!isSuggest && (
              <p className="text-sm text-gray-600 mb-4">
                デバイスの生体認証（指紋・顔認証）やセキュリティキーを使って、パスワードレスで安全にログインできるようになります。
              </p>
            )}
            <button
              onClick={handleRegister}
              disabled={loading}
              className="w-full py-3 bg-blue-600 text-white rounded font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? "登録中..." : "パスキーを登録"}
            </button>
            {isSuggest && (
              <button
                onClick={handleSkip}
                disabled={loading}
                className="w-full py-3 mt-3 text-gray-500 text-sm hover:text-gray-700 disabled:opacity-50"
              >
                あとで設定する
              </button>
            )}
          </div>
        )}

        {step === "done" && (
          <div className="text-center">
            <div className="text-green-500 text-5xl mb-4">&#10003;</div>
            <p className="text-lg font-medium text-gray-800 mb-2">
              パスキーが登録されました
            </p>
            <p className="text-sm text-gray-500">
              次回のログインからパスキーで認証できます。
            </p>
            {isSuggest && !redirectAfterLogin && (
              <a
                href="/"
                className="inline-block mt-4 text-sm text-blue-600 hover:underline"
              >
                トップに戻る
              </a>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
