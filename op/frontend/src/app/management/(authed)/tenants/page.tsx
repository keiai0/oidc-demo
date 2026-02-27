"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { tenantsApi } from "@/lib/api/tenants";
import { getErrorMessage } from "@/lib/fetcher";
import { queryKeys } from "@/lib/query/query-keys";
import { routes } from "@/lib/routes";
import { Alert } from "@/components/ui/alert";
import { DataTable } from "@/components/ui/data-table";
import { Loading } from "@/components/ui/loading";
import { PageHeader } from "@/components/ui/page-header";
import type { Tenant } from "@/types";

export default function TenantsPage() {
  const { data, isLoading, error } = useQuery({
    queryKey: queryKeys.tenants.list(),
    queryFn: () => tenantsApi.list(),
  });

  const columns = [
    {
      header: "Name",
      cell: (t: Tenant) => (
        <Link
          href={`${routes.management.tenantDetail(t.id)}`}
          className="text-blue-600 hover:underline font-medium"
        >
          {t.name}
        </Link>
      ),
    },
    {
      header: "Code",
      cell: (t: Tenant) => (
        <span className="text-gray-600 font-mono text-xs">{t.code}</span>
      ),
    },
    {
      header: "Created",
      cell: (t: Tenant) => (
        <span className="text-gray-500">
          {new Date(t.created_at).toLocaleDateString()}
        </span>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="Tenants"
        action={
          <Link
            href={routes.management.tenantNew}
            className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
          >
            Create Tenant
          </Link>
        }
      />

      {error && <Alert variant="error">{getErrorMessage(error)}</Alert>}

      {isLoading ? (
        <Loading />
      ) : (
        <>
          <DataTable
            columns={columns}
            data={data?.data ?? []}
            keyExtractor={(t) => t.id}
            emptyMessage="No tenants found."
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
