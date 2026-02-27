"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
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

export default function ClientsPage() {
  const searchParams = useSearchParams();
  const tenantId = searchParams.get("tenant_id") ?? "";

  const { data, isLoading, error } = useQuery({
    queryKey: queryKeys.clients.listByTenant(tenantId),
    queryFn: () => clientsApi.listByTenant(tenantId),
    enabled: !!tenantId,
  });

  const columns = [
    {
      header: "Name",
      cell: (c: Client) => (
        <Link
          href={`${routes.management.clientDetail(c.id)}`}
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
      header: "Auth Method",
      cell: (c: Client) => (
        <span className="text-gray-600">{c.token_endpoint_auth_method}</span>
      ),
    },
    {
      header: "PKCE",
      cell: (c: Client) =>
        c.require_pkce ? (
          <span className="text-green-600">Required</span>
        ) : (
          <span className="text-gray-400">Optional</span>
        ),
    },
    {
      header: "Status",
      cell: (c: Client) => (
        <Badge variant={c.status === "active" ? "active" : "inactive"}>
          {c.status}
        </Badge>
      ),
    },
  ];

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <Link
            href={`${routes.management.tenantDetail(tenantId)}`}
            className="text-sm text-blue-600 hover:underline"
          >
            &larr; Back to Tenant
          </Link>
          <h1 className="text-2xl font-bold text-gray-900 mt-1">Clients</h1>
        </div>
        <Link
          href={`${routes.management.tenantClientNew(tenantId)}`}
          className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
        >
          Create Client
        </Link>
      </div>

      {error && <Alert variant="error">{getErrorMessage(error)}</Alert>}

      {isLoading ? (
        <Loading />
      ) : (
        <>
          <DataTable
            columns={columns}
            data={data?.data ?? []}
            keyExtractor={(c) => c.id}
            emptyMessage="No clients found."
          />
          {data && data.total_count > 0 && (
            <p className="text-xs text-gray-400 mt-2">
              {data.total_count} total
            </p>
          )}
        </>
      )}
    </div>
  );
}
