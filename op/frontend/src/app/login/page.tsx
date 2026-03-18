"use client";

import { useState, useEffect } from "react";
import { Alert } from "@/components/ui/alert";
import {
  prepareRequestOptions,
  authenticationCredentialToJSON,
} from "@/lib/webauthn";

const API_URL = process.env.NEXT_PUBLIC_OP_BACKEND_BASE_URL || "http://localhost:8080";

export default function LoginPage() {
  const [loginId, setLoginId] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [passkeyLoading, setPasskeyLoading] = useState(false);
  const [tenantCode, setTenantCode] = useState("");
  const [redirectAfterLogin, setRedirectAfterLogin] = useState("");

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    setTenantCode(params.get("tenant_code") || "demo");
    setRedirectAfterLogin(params.get("redirect_after_login") || "");
  }, []);

  function handleLoginSuccess(redirectTo?: string) {
    if (redirectTo || redirectAfterLogin) {
      window.location.href = `${API_URL}${redirectTo || redirectAfterLogin}`;
    } else {
      window.location.href = "/account";
    }
  }

  // パスキーでログイン
  async function handlePasskeyLogin() {
    setError("");
    setPasskeyLoading(true);

    try {
      // 1. Begin: チャレンジ取得
      const beginRes = await fetch(`${API_URL}/internal/passkey/login/begin`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
      });

      if (!beginRes.ok) {
        setError("パスキー認証の開始に失敗しました");
        return;
      }

      const serverData = await beginRes.json();
      const challengeId = serverData.challenge_id;

      // 2. ブラウザの WebAuthn API を呼び出し
      const requestOptions = prepareRequestOptions(serverData);
      const credential = (await navigator.credentials.get(
        requestOptions
      )) as PublicKeyCredential | null;

      if (!credential) {
        setError("パスキー認証がキャンセルされました");
        return;
      }

      // 3. Complete: Assertion をサーバーに送信
      const credJSON = authenticationCredentialToJSON(credential);
      const completeUrl = new URL(`${API_URL}/internal/passkey/login/complete`);
      completeUrl.searchParams.set("challenge_id", challengeId);
      completeUrl.searchParams.set("tenant_code", tenantCode);

      const completeRes = await fetch(completeUrl.toString(), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(credJSON),
      });

      if (!completeRes.ok) {
        const data = await completeRes.json();
        if (data.error === "clone_detected") {
          setError("認証器のクローンが検出されました");
        } else if (data.error === "account_locked") {
          setError("アカウントがロックされています");
        } else if (
          data.error === "authentication_failed" &&
          data.error_description?.includes("no WebAuthn credentials")
        ) {
          setError(
            "パスキーが登録されていません。パスワードでログインし、パスキーを登録してください。"
          );
        } else {
          setError(data.error_description || "パスキー認証に失敗しました");
        }
        return;
      }

      // ログイン成功
      handleLoginSuccess(redirectAfterLogin);
    } catch (e) {
      if (e instanceof DOMException && e.name === "NotAllowedError") {
        setError("パスキー認証がキャンセルされました");
      } else if (
        e instanceof DOMException &&
        (e.name === "InvalidStateError" || e.name === "AbortError")
      ) {
        setError(
          "パスキーが見つかりませんでした。パスワードでログインし、パスキーを登録してください。"
        );
      } else {
        setError(
          "パスキーが見つかりませんでした。パスワードでログインし、パスキーを登録してください。"
        );
      }
    } finally {
      setPasskeyLoading(false);
    }
  }

  // パスワードでログイン
  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
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
        if (data.error === "invalid_credentials") {
          setError("ログインIDまたはパスワードが正しくありません");
        } else if (data.error === "account_locked") {
          setError("アカウントがロックされています。しばらく後に再試行してください。");
        } else if (data.error === "too_many_requests") {
          setError("リクエスト回数の上限に達しました。しばらく後に再試行してください。");
        } else {
          setError("ログインに失敗しました");
        }
        return;
      }

      const data = await res.json();

      if (data.mfa_setup_required) {
        const params = new URLSearchParams();
        params.set("tenant_code", tenantCode);
        if (redirectAfterLogin) {
          params.set("redirect_after_mfa", redirectAfterLogin);
        }
        window.location.href = `/mfa/setup?${params.toString()}`;
        return;
      }

      if (data.mfa_required) {
        const mfaParams = new URLSearchParams();
        mfaParams.set("tenant_code", tenantCode);
        if (redirectAfterLogin) {
          mfaParams.set("redirect_after_mfa", redirectAfterLogin);
        }
        if (data.mfa_methods) {
          mfaParams.set("mfa_methods", data.mfa_methods.join(","));
        }
        window.location.href = `/mfa/verify?${mfaParams.toString()}`;
        return;
      }

      // パスキー未登録 → 登録を提案（スキップ可能）
      if (!data.passkey_registered) {
        const suggestParams = new URLSearchParams();
        suggestParams.set("suggest", "1");
        if (redirectAfterLogin) {
          suggestParams.set("redirect_after_login", redirectAfterLogin);
        }
        window.location.href = `/mfa/setup/webauthn?${suggestParams.toString()}`;
        return;
      }

      handleLoginSuccess();
    } catch {
      setError("サーバーに接続できません");
    } finally {
      setLoading(false);
    }
  }

  const isLoading = loading || passkeyLoading;

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100">
      <div className="bg-white p-8 rounded-lg shadow-md w-full max-w-sm">
        <h1 className="text-2xl font-semibold text-center text-gray-800 mb-6">
          ログイン
        </h1>
        {error && <Alert variant="error">{error}</Alert>}
        <div className="mb-4 p-3 bg-gray-50 rounded border border-gray-200 text-xs text-gray-500">
          <p className="font-medium text-gray-600 mb-1">テスト用アカウント</p>
          <p>ログインID: <code className="bg-gray-100 px-1 rounded">testuser</code></p>
          <p>パスワード: <code className="bg-gray-100 px-1 rounded">password</code></p>
          <p>メール: <code className="bg-gray-100 px-1 rounded">testuser@example.com</code></p>
        </div>

        {/* パスキーでログイン */}
        <button
          onClick={handlePasskeyLogin}
          disabled={isLoading}
          className="w-full py-3 bg-gray-900 text-white rounded font-medium hover:bg-gray-800 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M2 18v3c0 .6.4 1 1 1h4v-3h3v-3h2l1.4-1.4a6.5 6.5 0 1 0-4-4Z"/><circle cx="16.5" cy="7.5" r=".5" fill="currentColor"/></svg>
          {passkeyLoading ? "認証中..." : "パスキーでログイン"}
        </button>

        <div className="relative my-5">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-gray-300" />
          </div>
          <div className="relative flex justify-center text-xs">
            <span className="bg-white px-2 text-gray-400">または</span>
          </div>
        </div>

        {/* パスワードでログイン */}
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
            disabled={isLoading}
            className="w-full py-3 bg-blue-600 text-white rounded font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? "ログイン中..." : "ログイン"}
          </button>
        </form>
        <div className="mt-4 text-center">
          <a
            href={`/password-reset-request?tenant_code=${tenantCode}`}
            className="text-sm text-blue-600 hover:underline"
          >
            パスワードをお忘れですか？
          </a>
        </div>
      </div>
    </div>
  );
}
