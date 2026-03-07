"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { clientsApi } from "@/lib/api/clients";
import { getErrorMessage } from "@/lib/fetcher";
import { queryKeys } from "@/lib/query/query-keys";
import { routes } from "@/lib/routes";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { DataTable } from "@/components/ui/data-table";
import { Loading } from "@/components/ui/loading";
import type { Client } from "@/types";

export default function AllClientsPage() {
  const {
    data: clientsData,
    isLoading,
    error,
  } = useQuery({
    queryKey: queryKeys.clients.list(250, 0),
    queryFn: () => clientsApi.listAll(250, 0),
  });

  const allClients = clientsData?.data ?? [];

  const columns = [
    {
      header: "名前",
      cell: (c: Client) => (
        <Link
          href={routes.management.clientDetail(c.id)}
          className="text-blue-600 hover:underline font-medium"
        >
          {c.name}
        </Link>
      ),
    },
    {
      header: "Client ID",
      cell: (c: Client) => (
        <span className="font-mono text-xs text-gray-600">{c.client_id}</span>
      ),
    },
    {
      header: "認証方式",
      cell: (c: Client) => (
        <span className="text-gray-600">{c.token_endpoint_auth_method}</span>
      ),
    },
    {
      header: "ステータス",
      cell: (c: Client) => (
        <Badge variant={c.status === "active" ? "active" : "inactive"}>
          {c.status}
        </Badge>
      ),
    },
  ];

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">クライアント</h1>

      {error && <Alert variant="error">{getErrorMessage(error)}</Alert>}

      {isLoading ? (
        <Loading />
      ) : (
        <>
          <DataTable
            columns={columns}
            data={allClients}
            keyExtractor={(c) => c.id}
            emptyMessage="クライアントがありません"
          />
          {allClients.length > 0 && (
            <p className="text-xs text-gray-400 mt-2">
              全 {allClients.length} 件
            </p>
          )}
        </>
      )}
    </div>
  );
}
