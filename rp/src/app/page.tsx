import Link from "next/link";
import { ClaimsConfig } from "@/components/claims-config";
import { SecurityDemoPanel } from "@/components/security-demo-panel";
import { isDemoMode } from "@/lib/env";

export default function LoginPage() {
  const demoMode = isDemoMode();

  return (
    <div className="flex items-center justify-center min-h-screen">
      <div className="w-full max-w-md space-y-6">
        <div className="p-8 bg-white rounded-lg shadow-md">
          <h1 className="text-2xl font-bold text-center mb-2">
            OIDC Relying Party
          </h1>
          <p className="text-gray-500 text-center mb-8">
            動作検証用 RP アプリケーション
          </p>

          <ClaimsConfig />

          <div className="mt-6 p-3 bg-gray-50 rounded border border-gray-200 text-xs text-gray-500">
            <p className="font-medium text-gray-600 mb-1">テスト用アカウント</p>
            <p>ログインID: <code className="bg-gray-100 px-1 rounded">testuser</code></p>
            <p>パスワード: <code className="bg-gray-100 px-1 rounded">password</code></p>
          </div>

          <p className="text-xs text-gray-400 text-center mt-4">
            OP の認可エンドポイントにリダイレクトし、
            <br />
            Authorization Code Flow (PKCE) で認証します
          </p>
        </div>

        <div className="p-4 bg-white rounded-lg shadow-md text-center">
          <Link
            href="/dynamic-registration"
            className="text-blue-600 hover:underline text-sm"
          >
            Dynamic Client Registration デモ →
          </Link>
        </div>

        {demoMode && <SecurityDemoPanel />}
      </div>
    </div>
  );
}
