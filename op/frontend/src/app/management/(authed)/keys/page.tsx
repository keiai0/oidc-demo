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
      setSuccess("Key rotated successfully");
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
      setSuccess("Key deactivated");
      setError("");
      queryClient.invalidateQueries({ queryKey: queryKeys.keys.all });
    },
    onError: (err) => {
      setError(getErrorMessage(err));
      setSuccess("");
    },
  });

  const handleRotate = () => {
    if (!confirm("Rotate signing key? The current key will be deactivated."))
      return;
    rotateMutation.mutate();
  };

  const handleDeactivate = (kid: string) => {
    if (!confirm(`Deactivate key ${kid}?`)) return;
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
      header: "Algorithm",
      cell: (k: SignKey) => (
        <span className="text-gray-600">{k.algorithm}</span>
      ),
    },
    {
      header: "Status",
      cell: (k: SignKey) => (
        <Badge variant={k.active ? "active" : "inactive"}>
          {k.active ? "active" : "inactive"}
        </Badge>
      ),
    },
    {
      header: "Created",
      cell: (k: SignKey) => (
        <span className="text-gray-500">
          {new Date(k.created_at).toLocaleString()}
        </span>
      ),
    },
    {
      header: "Actions",
      cell: (k: SignKey) =>
        k.active ? (
          <button
            onClick={() => handleDeactivate(k.kid)}
            className="text-red-600 hover:underline text-xs"
          >
            Deactivate
          </button>
        ) : null,
    },
  ];

  return (
    <div>
      <PageHeader
        title="Signing Keys"
        action={
          <button
            onClick={handleRotate}
            className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
          >
            Rotate Key
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
          emptyMessage="No signing keys found."
        />
      )}
    </div>
  );
}
