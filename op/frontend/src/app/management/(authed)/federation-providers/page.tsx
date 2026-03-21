"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { tenantsApi } from "@/lib/api/tenants";
import { federationProvidersApi } from "@/lib/api/federation-providers";
import { getErrorMessage } from "@/lib/fetcher";
import { queryKeys } from "@/lib/query/query-keys";
import { routes } from "@/lib/routes";
import { Alert } from "@/components/ui/alert";
import { DataTable } from "@/components/ui/data-table";
import { Loading } from "@/components/ui/loading";
import { PageHeader } from "@/components/ui/page-header";
import type { FederationProvider, Tenant } from "@/types";

export default function FederationProvidersPage() {
  const [selectedTenantId, setSelectedTenantId] = useState("");

  const { data: tenantsData } = useQuery({
    queryKey: queryKeys.tenants.list(),
    queryFn: () => tenantsApi.list(),
  });

  const tenants = tenantsData?.data ?? [];

  const { data, isLoading, error } = useQuery({
    queryKey: queryKeys.federationProviders.listByTenant(selectedTenantId),
    queryFn: () => federationProvidersApi.list(selectedTenantId),
    enabled: !!selectedTenantId,
  });

  // 最初のテナントを自動選択
  if (!selectedTenantId && tenants.length > 0) {
    setSelectedTenantId(tenants[0].id);
  }

  const columns = [
    {
      header: "名前",
      cell: (p: FederationProvider) => (
        <Link
          href={routes.management.federationProviderDetail(p.id)}
          className="text-blue-600 hover:underline font-medium"
        >
          {p.name}
        </Link>
      ),
    },
    {
      header: "Issuer",
      cell: (p: FederationProvider) => (
        <span className="text-gray-600 text-xs font-mono truncate max-w-xs block">
          {p.issuer}
        </span>
      ),
    },
    {
      header: "Client ID",
      cell: (p: FederationProvider) => (
        <span className="text-gray-600 text-xs font-mono">{p.client_id}</span>
      ),
    },
    {
      header: "JIT",
      cell: (p: FederationProvider) => (
        <span
          className={`text-xs px-2 py-0.5 rounded ${p.auto_provision ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-600"}`}
        >
          {p.auto_provision ? "有効" : "無効"}
        </span>
      ),
    },
    {
      header: "ステータス",
      cell: (p: FederationProvider) => (
        <span
          className={`text-xs px-2 py-0.5 rounded ${p.status === "active" ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`}
        >
          {p.status}
        </span>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="IdP 連携プロバイダ"
        action={
          selectedTenantId ? (
            <Link
              href={routes.management.federationProviderNew(selectedTenantId)}
              className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
            >
              プロバイダ追加
            </Link>
          ) : null
        }
      />

      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-600 mb-1">
          テナント
        </label>
        <select
          value={selectedTenantId}
          onChange={(e) => setSelectedTenantId(e.target.value)}
          className="px-3 py-2 border border-gray-300 rounded text-sm"
        >
          <option value="">テナントを選択</option>
          {tenants.map((t: Tenant) => (
            <option key={t.id} value={t.id}>
              {t.name} ({t.code})
            </option>
          ))}
        </select>
      </div>

      {error && <Alert variant="error">{getErrorMessage(error)}</Alert>}

      {!selectedTenantId ? (
        <p className="text-gray-500 text-sm">テナントを選択してください</p>
      ) : isLoading ? (
        <Loading />
      ) : (
        <DataTable
          columns={columns}
          data={data?.data ?? []}
          keyExtractor={(p) => p.id}
          emptyMessage="連携プロバイダがありません"
        />
      )}
    </div>
  );
}
