"use client";

import { useEffect, useState, Suspense } from "react";
import { useSearchParams } from "next/navigation";
import { Alert } from "@/components/ui/alert";

const API_URL = process.env.NEXT_PUBLIC_OP_BACKEND_BASE_URL || "http://localhost:8080";

function EmailVerifyContent() {
  const searchParams = useSearchParams();
  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");
  const [errorMessage, setErrorMessage] = useState("");

  useEffect(() => {
    const token = searchParams.get("token");
    if (!token) {
      setErrorMessage("トークンが見つかりません");
      setStatus("error");
      return;
    }

    fetch(`${API_URL}/internal/email/verify`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ token }),
    })
      .then(async (res) => {
        if (!res.ok) {
          const data = await res.json();
          if (data.error === "token_invalid") {
            setErrorMessage(
              "トークンが無効または期限切れです。再度メールアドレス変更を申請してください。"
            );
          } else {
            setErrorMessage("メールアドレスの確認に失敗しました");
          }
          setStatus("error");
        } else {
          setStatus("success");
        }
      })
      .catch(() => {
        setErrorMessage("サーバーに接続できません");
        setStatus("error");
      });
  }, [searchParams]);

  if (status === "loading") {
    return (
      <div className="bg-white p-8 rounded-lg shadow-md w-full max-w-sm text-center">
        <p className="text-gray-600">確認中...</p>
      </div>
    );
  }

  if (status === "success") {
    return (
      <div className="bg-white p-8 rounded-lg shadow-md w-full max-w-sm">
        <h1 className="text-2xl font-semibold text-center text-gray-800 mb-4">
          メールアドレスが変更されました
        </h1>
        <p className="text-sm text-gray-600 text-center">
          新しいメールアドレスへの変更が完了しました。
        </p>
      </div>
    );
  }

  return (
    <div className="bg-white p-8 rounded-lg shadow-md w-full max-w-sm">
      <h1 className="text-2xl font-semibold text-center text-gray-800 mb-4">
        確認に失敗しました
      </h1>
      <Alert variant="error">{errorMessage}</Alert>
    </div>
  );
}

export default function EmailVerifyPage() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100">
      <Suspense
        fallback={
          <div className="bg-white p-8 rounded-lg shadow-md w-full max-w-sm text-center">
            <p className="text-gray-600">読み込み中...</p>
          </div>
        }
      >
        <EmailVerifyContent />
      </Suspense>
    </div>
  );
}
