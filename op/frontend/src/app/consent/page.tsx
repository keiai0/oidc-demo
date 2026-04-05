"use client";

import { useState, useEffect } from "react";
import { Alert } from "@/components/ui/alert";

const API_URL = process.env.NEXT_PUBLIC_OP_BACKEND_BASE_URL || "http://localhost:8080";

const scopeDescriptions: Record<string, string> = {
  openid: "基本的な認証情報",
  profile: "プロフィール情報（名前など）",
  email: "メールアドレス",
  offline_access: "オフラインアクセス（長期間のトークン更新）",
};

// RFC 9396 の標準フィールド名の日本語ラベル
const rarFieldLabels: Record<string, string> = {
  actions: "操作",
  locations: "リソースの場所",
  identifier: "識別子",
  datatypes: "データ種別",
  privileges: "権限",
};

// RFC 9396 の標準フィールド（type 以外）。これ以外は追加フィールドとして表示
const rarStandardFields = new Set(["type", "actions", "locations", "identifier", "datatypes", "privileges"]);

// authorization_details の1要素の型
interface AuthorizationDetail {
  type: string;
  actions?: string[];
  locations?: string[];
  identifier?: string;
  datatypes?: string[];
  privileges?: string[];
  [key: string]: unknown;
}

export default function ConsentPage() {
  const [clientId, setClientId] = useState("");
  const [clientName, setClientName] = useState("");
  const [scopes, setScopes] = useState<string[]>([]);
  const [authorizationDetails, setAuthorizationDetails] = useState<AuthorizationDetail[]>([]);
  const [redirectAfterConsent, setRedirectAfterConsent] = useState("");
  const [redirectURI, setRedirectURI] = useState("");
  const [state, setState] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    setClientName(params.get("client_name") || "不明なアプリケーション");
    setScopes((params.get("scope") || "openid").split(" "));
    setRedirectAfterConsent(params.get("redirect_after_consent") || "");

    // client_id はバックエンドに送るため内部 UUID が必要
    // authorize.go から渡される client_id は外部 ID
    // consent handler では client_id を UUID としてパースするため
    // ここでは DB の内部 ID を取得する必要があるが、
    // 現在の設計では authorize.go が外部 client_id を渡すのみ。
    // → consent handler 側で外部 client_id → UUID 解決が必要。
    // 暫定: client_id パラメータをそのまま使う
    setClientId(params.get("client_id") || "");
    setRedirectURI(params.get("redirect_uri") || "");
    setState(params.get("state") || "");

    // RFC 9396: authorization_details は URL エンコードされた JSON 配列
    const adParam = params.get("authorization_details");
    if (adParam) {
      try {
        const parsed = JSON.parse(adParam);
        if (Array.isArray(parsed)) {
          setAuthorizationDetails(parsed);
        }
      } catch {
        // パース失敗時は無視（authorization_details なしとして扱う）
      }
    }
  }, []);

  async function handleConsent(approved: boolean) {
    setError("");
    setLoading(true);

    try {
      const res = await fetch(`${API_URL}/internal/consent`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          client_id: clientId,
          scopes,
          approved,
          redirect_after_consent: redirectAfterConsent,
          redirect_uri: redirectURI,
          state,
        }),
      });

      if (!res.ok) {
        const data = await res.json();
        if (data.error === "unauthorized") {
          setError("セッションが無効です。再度ログインしてください。");
        } else {
          setError("処理に失敗しました");
        }
        return;
      }

      const data = await res.json();
      if (data.redirect_to) {
        // 承認: authorize URL にリダイレクト
        // 拒否: redirect_uri に error=access_denied でリダイレクト
        if (data.redirect_to.startsWith("/")) {
          window.location.href = `${API_URL}${data.redirect_to}`;
        } else {
          window.location.href = data.redirect_to;
        }
      }
    } catch {
      setError("サーバーに接続できません");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100">
      <div className="bg-white p-8 rounded-lg shadow-md w-full max-w-md">
        <h1 className="text-2xl font-semibold text-center text-gray-800 mb-2">
          アクセス許可
        </h1>
        <p className="text-sm text-gray-600 text-center mb-6">
          <span className="font-medium text-gray-800">{clientName}</span>{" "}
          が以下の情報へのアクセスを求めています。
        </p>

        {error && <Alert variant="error">{error}</Alert>}

        <div className="mb-6 space-y-3">
          {scopes.map((scope) => (
            <div
              key={scope}
              className="flex items-center p-3 bg-gray-50 rounded border border-gray-200"
            >
              <svg
                className="w-5 h-5 text-blue-500 mr-3 flex-shrink-0"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
              <div>
                <p className="text-sm font-medium text-gray-700">
                  {scopeDescriptions[scope] || scope}
                </p>
                <p className="text-xs text-gray-400">{scope}</p>
              </div>
            </div>
          ))}
        </div>

        {authorizationDetails.length > 0 && (
          <div className="mb-6 space-y-3">
            <p className="text-xs text-gray-500 font-medium uppercase tracking-wide">
              詳細な認可要求（Rich Authorization Requests）
            </p>
            {authorizationDetails.map((detail, index) => (
              <div
                key={index}
                className="p-3 bg-amber-50 rounded border border-amber-200"
              >
                <div className="flex items-center mb-2">
                  <svg
                    className="w-5 h-5 text-amber-500 mr-3 flex-shrink-0"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                    />
                  </svg>
                  <p className="text-sm font-medium text-gray-700">
                    {detail.type}
                  </p>
                </div>
                <div className="ml-8 space-y-1">
                  {/* 標準フィールド（type 以外） */}
                  {(["actions", "locations", "datatypes", "privileges"] as const).map(
                    (field) => {
                      const value = detail[field];
                      if (!value || !Array.isArray(value) || value.length === 0) return null;
                      return (
                        <div key={field} className="text-xs text-gray-600">
                          <span className="font-medium">{rarFieldLabels[field]}:</span>{" "}
                          {value.join(", ")}
                        </div>
                      );
                    }
                  )}
                  {detail.identifier && (
                    <div className="text-xs text-gray-600">
                      <span className="font-medium">{rarFieldLabels.identifier}:</span>{" "}
                      {detail.identifier}
                    </div>
                  )}
                  {/* 追加フィールド（標準以外） */}
                  {Object.entries(detail)
                    .filter(([key]) => !rarStandardFields.has(key))
                    .map(([key, value]) => (
                      <div key={key} className="text-xs text-gray-600">
                        <span className="font-medium">{key}:</span>{" "}
                        {typeof value === "object"
                          ? JSON.stringify(value)
                          : String(value)}
                      </div>
                    ))}
                </div>
              </div>
            ))}
          </div>
        )}

        <div className="flex gap-3">
          <button
            onClick={() => handleConsent(false)}
            disabled={loading}
            className="flex-1 py-3 border border-gray-300 text-gray-700 rounded font-medium hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            拒否
          </button>
          <button
            onClick={() => handleConsent(true)}
            disabled={loading}
            className="flex-1 py-3 bg-blue-600 text-white rounded font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? "処理中..." : "許可する"}
          </button>
        </div>
      </div>
    </div>
  );
}
