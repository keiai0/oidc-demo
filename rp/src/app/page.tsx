import Link from "next/link";

export default function LoginPage() {
  return (
    <div className="flex items-center justify-center min-h-screen">
      <div className="w-full max-w-md p-8 bg-white rounded-lg shadow-md">
        <h1 className="text-2xl font-bold text-center mb-2">
          OIDC Relying Party
        </h1>
        <p className="text-gray-500 text-center mb-8">
          動作検証用 RP アプリケーション
        </p>

        <Link
          href="/api/auth/login"
          className="block w-full py-3 px-4 bg-blue-600 text-white text-center rounded-lg hover:bg-blue-700 transition-colors font-medium"
        >
          OP でログイン
        </Link>

        <p className="text-xs text-gray-400 text-center mt-6">
          OP の認可エンドポイントにリダイレクトし、
          <br />
          Authorization Code Flow (PKCE) で認証します
        </p>
      </div>
    </div>
  );
}
