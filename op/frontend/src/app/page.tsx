import Link from "next/link";

export default function Home() {
  return (
    <main className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="w-full max-w-sm space-y-6">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-gray-900">
            OpenID Provider
          </h1>
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
