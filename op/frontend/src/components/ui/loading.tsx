"use client";

export function Loading({ message = "Loading..." }: { message?: string }) {
  return <p className="text-gray-500">{message}</p>;
}

export function FullScreenLoading() {
  return (
    <div className="flex h-screen items-center justify-center">
      <p className="text-gray-500">Loading...</p>
    </div>
  );
}
