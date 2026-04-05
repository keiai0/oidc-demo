"use client";

import { useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { tenantsApi } from "@/lib/api/tenants";
import { authorizationDetailTypesApi } from "@/lib/api/authorization-detail-types";
import { getErrorMessage } from "@/lib/fetcher";
import { queryKeys } from "@/lib/query/query-keys";
import { routes } from "@/lib/routes";
import { Alert } from "@/components/ui/alert";
import { DataTable } from "@/components/ui/data-table";
import { Loading } from "@/components/ui/loading";
import { PageHeader } from "@/components/ui/page-header";
import type { AuthorizationDetailType, Tenant } from "@/types";

export default function AuthorizationDetailTypesPage() {
  const [selectedTenantId, setSelectedTenantId] = useState("");

  const { data: tenantsData } = useQuery({
    queryKey: queryKeys.tenants.list(),
    queryFn: () => tenantsApi.list(),
  });

  const tenants = tenantsData?.data ?? [];

  const { data, isLoading, error } = useQuery({
    queryKey: queryKeys.authorizationDetailTypes.listByTenant(selectedTenantId),
    queryFn: () => authorizationDetailTypesApi.list(selectedTenantId),
    enabled: !!selectedTenantId,
  });

  // 最初のテナントを自動選択
  if (!selectedTenantId && tenants.length > 0) {
    setSelectedTenantId(tenants[0].id);
  }

  const columns = [
    {
      header: "タイプ名",
      cell: (t: AuthorizationDetailType) => (
        <Link
          href={routes.management.authorizationDetailTypeDetail(t.id)}
          className="text-blue-600 hover:underline font-medium"
        >
          {t.type_name}
        </Link>
      ),
    },
    {
      header: "説明",
      cell: (t: AuthorizationDetailType) => (
        <span className="text-gray-600 text-sm">{t.description}</span>
      ),
    },
    {
      header: "許可アクション数",
      cell: (t: AuthorizationDetailType) => (
        <span className="text-sm">{t.allowed_actions.length}</span>
      ),
    },
    {
      header: "許可ロケーション数",
      cell: (t: AuthorizationDetailType) => (
        <span className="text-sm">{t.allowed_locations.length}</span>
      ),
    },
  ];

  return (
    <div>
      <PageHeader
        title="認可詳細タイプ"
        action={
          selectedTenantId ? (
            <Link
              href={routes.management.authorizationDetailTypeNew(selectedTenantId)}
              className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
            >
              タイプ追加
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
          keyExtractor={(t) => t.id}
          emptyMessage="認可詳細タイプがありません"
        />
      )}
    </div>
  );
}
