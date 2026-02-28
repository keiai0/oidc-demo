"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { keysApi } from "@/lib/api/keys";
import { getErrorMessage } from "@/lib/fetcher";
import { queryKeys } from "@/lib/query/query-keys";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { DataTable } from "@/components/ui/data-table";
import { Loading } from "@/components/ui/loading";
import { PageHeader } from "@/components/ui/page-header";
import type { SignKey } from "@/types";

export default function KeysPage() {
  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const { data: keys, isLoading } = useQuery({
    queryKey: queryKeys.keys.list(),
    queryFn: keysApi.list,
  });

  const rotateMutation = useMutation({
    mutationFn: keysApi.rotate,
    onSuccess: () => {
      setSuccess("署名鍵をローテーションしました");
      setError("");
      queryClient.invalidateQueries({ queryKey: queryKeys.keys.all });
    },
    onError: (err) => {
      setError(getErrorMessage(err));
      setSuccess("");
    },
  });

  const deactivateMutation = useMutation({
    mutationFn: (kid: string) => keysApi.deactivate(kid),
    onSuccess: () => {
      setSuccess("署名鍵を無効化しました");
      setError("");
      queryClient.invalidateQueries({ queryKey: queryKeys.keys.all });
    },
    onError: (err) => {
      setError(getErrorMessage(err));
      setSuccess("");
    },
  });

  const handleRotate = () => {
    if (!confirm("署名鍵をローテーションしますか？現在の鍵は無効化されます。"))
      return;
    rotateMutation.mutate();
  };

  const handleDeactivate = (kid: string) => {
    if (!confirm(`署名鍵 ${kid} を無効化しますか？`)) return;
    deactivateMutation.mutate(kid);
  };

  const columns = [
    {
      header: "KID",
      cell: (k: SignKey) => (
        <span className="font-mono text-xs">{k.kid}</span>
      ),
    },
    {
      header: "アルゴリズム",
      cell: (k: SignKey) => (
        <span className="text-gray-600">{k.algorithm}</span>
      ),
    },
    {
      header: "ステータス",
      cell: (k: SignKey) => (
        <Badge variant={k.active ? "active" : "inactive"}>
          {k.active ? "有効" : "無効"}
        </Badge>
      ),
    },
    {
      header: "作成日時",
      cell: (k: SignKey) => (
        <span className="text-gray-500">
          {new Date(k.created_at).toLocaleString()}
        </span>
      ),
    },
    {
      header: "操作",
      cell: (k: SignKey) =>
        k.active ? (
          <button
            onClick={() => handleDeactivate(k.kid)}
            className="text-red-600 hover:underline text-xs"
          >
            無効化
          </button>
        ) : null,
    },
  ];

  return (
    <div>
      <PageHeader
        title="署名鍵"
        action={
          <button
            onClick={handleRotate}
            className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
          >
            鍵のローテーション
          </button>
        }
      />

      {error && <Alert variant="error">{error}</Alert>}
      {success && <Alert variant="success">{success}</Alert>}

      {isLoading ? (
        <Loading />
      ) : (
        <DataTable
          columns={columns}
          data={keys ?? []}
          keyExtractor={(k) => k.kid}
          emptyMessage="署名鍵がありません"
        />
      )}
    </div>
  );
}
