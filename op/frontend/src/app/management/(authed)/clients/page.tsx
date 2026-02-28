"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { tenantsApi } from "@/lib/api/tenants";
import { clientsApi } from "@/lib/api/clients";
import { getErrorMessage } from "@/lib/fetcher";
import { queryKeys } from "@/lib/query/query-keys";
import { routes } from "@/lib/routes";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { DataTable } from "@/components/ui/data-table";
import { Loading } from "@/components/ui/loading";
import type { Client, Tenant } from "@/types";

export default function AllClientsPage() {
  const {
    data: tenantsData,
    isLoading: tenantsLoading,
    error: tenantsError,
  } = useQuery({
    queryKey: queryKeys.tenants.list(250, 0),
    queryFn: () => tenantsApi.list(250, 0),
  });

  const tenants = tenantsData?.data ?? [];

  const clientQueries = tenants.map((t) => ({
    queryKey: queryKeys.clients.listByTenant(t.id),
    queryFn: () => clientsApi.listByTenant(t.id),
    enabled: tenants.length > 0,
  }));

  // テナントごとにクライアントを取得して集約する
  const {
    data: allClients,
    isLoading: clientsLoading,
    error: clientsError,
  } = useQuery({
    queryKey: ["clients", "all-tenants", tenants.map((t) => t.id)],
    queryFn: async () => {
      const results = await Promise.all(
        tenants.map((t) => clientsApi.listByTenant(t.id)),
      );
      return results.flatMap((r) => r.data);
    },
    enabled: tenants.length > 0,
  });

  const tenantMap = new Map<string, Tenant>(tenants.map((t) => [t.id, t]));

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
      header: "テナント",
      cell: (c: Client) => {
        const tenant = tenantMap.get(c.tenant_id);
        return tenant ? (
          <Link
            href={routes.management.tenantDetail(c.tenant_id)}
            className="text-blue-600 hover:underline"
          >
            {tenant.name}
          </Link>
        ) : (
          <span className="text-gray-400">-</span>
        );
      },
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

  const isLoading = tenantsLoading || clientsLoading;
  const error = tenantsError || clientsError;

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
            data={allClients ?? []}
            keyExtractor={(c) => c.id}
            emptyMessage="クライアントがありません"
          />
          {allClients && allClients.length > 0 && (
            <p className="text-xs text-gray-400 mt-2">
              全 {allClients.length} 件
            </p>
          )}
        </>
      )}
    </div>
  );
}
