"use client";

export function Loading({ message = "読み込み中..." }: { message?: string }) {
  return <p className="text-gray-500">{message}</p>;
}

export function FullScreenLoading() {
  return (
    <div className="flex h-screen items-center justify-center">
      <p className="text-gray-500">読み込み中...</p>
    </div>
  );
}
