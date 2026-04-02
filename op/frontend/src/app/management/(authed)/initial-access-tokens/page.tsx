"use client";

import { useState } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { initialAccessTokensApi } from "@/lib/api/initial-access-tokens";
import { tenantsApi } from "@/lib/api/tenants";
import { getErrorMessage } from "@/lib/fetcher";
import { queryKeys } from "@/lib/query/query-keys";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { DataTable } from "@/components/ui/data-table";
import { Loading } from "@/components/ui/loading";
import { PageHeader } from "@/components/ui/page-header";
import type { InitialAccessToken } from "@/types/initial-access-token";
import type { Tenant } from "@/types";

export default function InitialAccessTokensPage() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const queryClient = useQueryClient();

  const tenantId = searchParams.get("tenant_id") ?? "";
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [createdToken, setCreatedToken] = useState("");
  const [maxRegistrations, setMaxRegistrations] = useState(0);
  const [expiresInHours, setExpiresInHours] = useState(24);
  const [showForm, setShowForm] = useState(false);

  // テナント一覧を取得してセレクタ表示
  const { data: tenantsData } = useQuery({
    queryKey: queryKeys.tenants.list(),
    queryFn: () => tenantsApi.list(),
  });

  const tenants: Tenant[] = tenantsData?.data ?? [];

  // IAT 一覧
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.initialAccessTokens.listByTenant(tenantId),
    queryFn: () => initialAccessTokensApi.list(tenantId),
    enabled: !!tenantId,
  });

  // IAT 発行
  const createMutation = useMutation({
    mutationFn: () =>
      initialAccessTokensApi.create(tenantId, {
        max_registrations: maxRegistrations,
        expires_in: expiresInHours * 3600,
      }),
    onSuccess: (resp) => {
      setCreatedToken(resp.token);
      setSuccess("Initial Access Token を発行しました（以下のトークンは一度だけ表示されます）");
      setError("");
      setShowForm(false);
      queryClient.invalidateQueries({
        queryKey: queryKeys.initialAccessTokens.all,
      });
    },
    onError: (err) => {
      setError(getErrorMessage(err));
      setSuccess("");
    },
  });

  // IAT 無効化
  const revokeMutation = useMutation({
    mutationFn: (id: string) => initialAccessTokensApi.revoke(id),
    onSuccess: () => {
      setSuccess("Initial Access Token を無効化しました");
      setError("");
      setCreatedToken("");
      queryClient.invalidateQueries({
        queryKey: queryKeys.initialAccessTokens.all,
      });
    },
    onError: (err) => {
      setError(getErrorMessage(err));
      setSuccess("");
    },
  });

  const handleTenantChange = (newTenantId: string) => {
    setCreatedToken("");
    setError("");
    setSuccess("");
    router.push(`/management/initial-access-tokens?tenant_id=${newTenantId}`);
  };

  const handleRevoke = (id: string) => {
    if (!confirm("この Initial Access Token を無効化しますか？")) return;
    revokeMutation.mutate(id);
  };

  const getStatus = (iat: InitialAccessToken): { label: string; variant: "active" | "inactive" } => {
    if (iat.revoked_at) return { label: "無効化済み", variant: "inactive" };
    if (new Date(iat.expires_at) < new Date()) return { label: "期限切れ", variant: "inactive" };
    if (iat.max_registrations > 0 && iat.used_count >= iat.max_registrations)
      return { label: "使用上限到達", variant: "inactive" };
    return { label: "有効", variant: "active" };
  };

  const columns = [
    {
      header: "ID",
      cell: (t: InitialAccessToken) => (
        <span className="font-mono text-xs">{t.id.slice(0, 8)}...</span>
      ),
    },
    {
      header: "ステータス",
      cell: (t: InitialAccessToken) => {
        const s = getStatus(t);
        return <Badge variant={s.variant}>{s.label}</Badge>;
      },
    },
    {
      header: "使用回数",
      cell: (t: InitialAccessToken) => (
        <span className="text-gray-600">
          {t.used_count} / {t.max_registrations === 0 ? "∞" : t.max_registrations}
        </span>
      ),
    },
    {
      header: "有効期限",
      cell: (t: InitialAccessToken) => (
        <span className="text-gray-500 text-xs">
          {new Date(t.expires_at).toLocaleString()}
        </span>
      ),
    },
    {
      header: "作成日時",
      cell: (t: InitialAccessToken) => (
        <span className="text-gray-500 text-xs">
          {new Date(t.created_at).toLocaleString()}
        </span>
      ),
    },
    {
      header: "操作",
      cell: (t: InitialAccessToken) => {
        const s = getStatus(t);
        return s.variant === "active" ? (
          <button
            onClick={() => handleRevoke(t.id)}
            className="text-red-600 hover:underline text-xs"
          >
            無効化
          </button>
        ) : null;
      },
    },
  ];

  return (
    <div>
      <PageHeader
        title="Initial Access Token 管理"
        action={
          tenantId ? (
            <button
              onClick={() => setShowForm(!showForm)}
              className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
            >
              {showForm ? "キャンセル" : "新規発行"}
            </button>
          ) : null
        }
      />

      {/* テナントセレクタ */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-700 mb-1">
          テナント
        </label>
        <select
          value={tenantId}
          onChange={(e) => handleTenantChange(e.target.value)}
          className="w-full max-w-md border border-gray-300 rounded px-3 py-2 text-sm"
        >
          <option value="">テナントを選択してください</option>
          {tenants.map((t) => (
            <option key={t.id} value={t.id}>
              {t.name} ({t.code})
            </option>
          ))}
        </select>
      </div>

      {error && <Alert variant="error">{error}</Alert>}
      {success && <Alert variant="success">{success}</Alert>}

      {/* 発行されたトークン表示 */}
      {createdToken && (
        <div className="mb-4 p-4 bg-yellow-50 border border-yellow-200 rounded">
          <p className="text-sm font-medium text-yellow-800 mb-2">
            発行されたトークン（この画面を離れると二度と表示されません）
          </p>
          <code className="block p-2 bg-white border rounded text-xs break-all select-all">
            {createdToken}
          </code>
        </div>
      )}

      {/* 発行フォーム */}
      {showForm && tenantId && (
        <div className="mb-6 p-4 bg-gray-50 border rounded">
          <h3 className="text-sm font-medium mb-3">新規 Initial Access Token</h3>
          <div className="grid grid-cols-2 gap-4 max-w-lg">
            <div>
              <label className="block text-xs text-gray-600 mb-1">
                最大登録回数（0 = 無制限）
              </label>
              <input
                type="number"
                min={0}
                value={maxRegistrations}
                onChange={(e) => setMaxRegistrations(Number(e.target.value))}
                className="w-full border border-gray-300 rounded px-3 py-2 text-sm"
              />
            </div>
            <div>
              <label className="block text-xs text-gray-600 mb-1">
                有効期間（時間）
              </label>
              <input
                type="number"
                min={1}
                value={expiresInHours}
                onChange={(e) => setExpiresInHours(Number(e.target.value))}
                className="w-full border border-gray-300 rounded px-3 py-2 text-sm"
              />
            </div>
          </div>
          <button
            onClick={() => createMutation.mutate()}
            disabled={createMutation.isPending}
            className="mt-3 px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700 disabled:opacity-50"
          >
            {createMutation.isPending ? "発行中..." : "発行する"}
          </button>
        </div>
      )}

      {/* IAT 一覧 */}
      {!tenantId ? (
        <p className="text-gray-500 text-sm">
          テナントを選択すると IAT 一覧が表示されます
        </p>
      ) : isLoading ? (
        <Loading />
      ) : (
        <DataTable
          columns={columns}
          data={data?.data ?? []}
          keyExtractor={(t) => t.id}
          emptyMessage="Initial Access Token がありません"
        />
      )}
    </div>
  );
}
