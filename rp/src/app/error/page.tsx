"use client";

import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { Suspense } from "react";

function ErrorContent() {
  const searchParams = useSearchParams();
  const error = searchParams.get("error") ?? "unknown_error";
  const errorDescription =
    searchParams.get("error_description") ?? "不明なエラーが発生しました";

  return (
    <div className="flex items-center justify-center min-h-screen">
      <div className="w-full max-w-md p-8 bg-white rounded-lg shadow-md">
        <h1 className="text-2xl font-bold text-red-600 mb-4">認証エラー</h1>

        <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
          <p className="text-sm font-mono text-red-800 mb-2">
            エラーコード: {error}
          </p>
          <p className="text-sm text-red-700">{errorDescription}</p>
        </div>

        <Link
          href="/"
          className="block w-full py-3 px-4 bg-gray-600 text-white text-center rounded-lg hover:bg-gray-700 transition-colors font-medium"
        >
          ログインページに戻る
        </Link>
      </div>
    </div>
  );
}

export default function ErrorPage() {
  return (
    <Suspense
      fallback={
        <div className="flex items-center justify-center min-h-screen">
          <p className="text-gray-500">読み込み中...</p>
        </div>
      }
    >
      <ErrorContent />
    </Suspense>
  );
}
