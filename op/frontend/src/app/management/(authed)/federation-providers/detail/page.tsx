"use client";

import { useState } from "react";
import { useSearchParams } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { federationProvidersApi } from "@/lib/api/federation-providers";
import { getErrorMessage } from "@/lib/fetcher";
import { queryKeys } from "@/lib/query/query-keys";
import { routes } from "@/lib/routes";
import { Alert } from "@/components/ui/alert";
import { Loading } from "@/components/ui/loading";
import { PageHeader } from "@/components/ui/page-header";

export default function FederationProviderDetailPage() {
  const searchParams = useSearchParams();
  const id = searchParams.get("id") || "";
  const queryClient = useQueryClient();

  const [editing, setEditing] = useState(false);
  const [issuer, setIssuer] = useState("");
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [scopes, setScopes] = useState("");
  const [autoProvision, setAutoProvision] = useState(true);
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const { data: provider, isLoading } = useQuery({
    queryKey: queryKeys.federationProviders.detail(id),
    queryFn: () => federationProvidersApi.get(id),
    enabled: !!id,
  });

  function startEdit() {
    if (!provider) return;
    setIssuer(provider.issuer);
    setClientId(provider.client_id);
    setClientSecret("");
    setScopes(provider.scopes);
    setAutoProvision(provider.auto_provision);
    setStatus(provider.status);
    setEditing(true);
    setError("");
    setSuccess("");
  }

  const updateMutation = useMutation({
    mutationFn: () => {
      const body: Record<string, unknown> = {
        issuer,
        client_id: clientId,
        scopes,
        auto_provision: autoProvision,
        status,
      };
      if (clientSecret) {
        body.client_secret = clientSecret;
      }
      return federationProvidersApi.update(id, body as Parameters<typeof federationProvidersApi.update>[1]);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.federationProviders.detail(id),
      });
      setEditing(false);
      setSuccess("更新しました");
    },
    onError: (err) => setError(getErrorMessage(err)),
  });

  const deleteMutation = useMutation({
    mutationFn: () => federationProvidersApi.delete(id),
    onSuccess: () => {
      window.location.href = routes.management.federationProviders;
    },
    onError: (err) => setError(getErrorMessage(err)),
  });

  if (isLoading) return <Loading />;
  if (!provider) return <Alert variant="error">プロバイダが見つかりません</Alert>;

  const inputClass =
    "w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent";

  return (
    <div>
      <PageHeader
        title={`IdP 連携: ${provider.name}`}
        action={
          !editing ? (
            <div className="flex gap-2">
              <button
                onClick={startEdit}
                className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
              >
                編集
              </button>
              <button
                onClick={() => {
                  if (confirm("このプロバイダを削除しますか？")) {
                    deleteMutation.mutate();
                  }
                }}
                className="px-4 py-2 bg-red-600 text-white text-sm rounded hover:bg-red-700"
              >
                削除
              </button>
            </div>
          ) : null
        }
      />

      {error && <Alert variant="error" className="mb-4">{error}</Alert>}
      {success && <Alert variant="success" className="mb-4">{success}</Alert>}

      <div className="bg-white border border-gray-200 rounded-lg p-6 max-w-lg space-y-4">
        {editing ? (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              setError("");
              updateMutation.mutate();
            }}
            className="space-y-4"
          >
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1">Issuer URL</label>
              <input type="url" value={issuer} onChange={(e) => setIssuer(e.target.value)} className={inputClass} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1">Client ID</label>
              <input type="text" value={clientId} onChange={(e) => setClientId(e.target.value)} className={inputClass} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1">Client Secret (変更する場合のみ)</label>
              <input type="password" value={clientSecret} onChange={(e) => setClientSecret(e.target.value)} placeholder="変更しない場合は空欄" className={inputClass} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1">Scopes</label>
              <input type="text" value={scopes} onChange={(e) => setScopes(e.target.value)} className={inputClass} />
            </div>
            <div className="flex items-center gap-2">
              <input type="checkbox" id="autoProvision" checked={autoProvision} onChange={(e) => setAutoProvision(e.target.checked)} className="rounded" />
              <label htmlFor="autoProvision" className="text-sm text-gray-600">JIT プロビジョニング</label>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1">ステータス</label>
              <select value={status} onChange={(e) => setStatus(e.target.value)} className={inputClass}>
                <option value="active">active</option>
                <option value="disabled">disabled</option>
              </select>
            </div>
            <div className="flex gap-2 pt-2">
              <button type="button" onClick={() => setEditing(false)} className="px-4 py-2 border border-gray-300 rounded text-sm hover:bg-gray-50">キャンセル</button>
              <button type="submit" disabled={updateMutation.isPending} className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700 disabled:opacity-50">
                {updateMutation.isPending ? "保存中..." : "保存"}
              </button>
            </div>
          </form>
        ) : (
          <dl className="space-y-3">
            <div>
              <dt className="text-xs text-gray-500">名前</dt>
              <dd className="font-medium">{provider.name}</dd>
            </div>
            <div>
              <dt className="text-xs text-gray-500">Issuer URL</dt>
              <dd className="text-sm font-mono break-all">{provider.issuer}</dd>
            </div>
            <div>
              <dt className="text-xs text-gray-500">Client ID</dt>
              <dd className="text-sm font-mono">{provider.client_id}</dd>
            </div>
            <div>
              <dt className="text-xs text-gray-500">Scopes</dt>
              <dd className="text-sm">{provider.scopes}</dd>
            </div>
            <div>
              <dt className="text-xs text-gray-500">JIT プロビジョニング</dt>
              <dd>{provider.auto_provision ? "有効" : "無効"}</dd>
            </div>
            <div>
              <dt className="text-xs text-gray-500">ステータス</dt>
              <dd>
                <span className={`text-xs px-2 py-0.5 rounded ${provider.status === "active" ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`}>
                  {provider.status}
                </span>
              </dd>
            </div>
            <div>
              <dt className="text-xs text-gray-500">作成日時</dt>
              <dd className="text-sm text-gray-600">{new Date(provider.created_at).toLocaleString()}</dd>
            </div>
          </dl>
        )}
      </div>
    </div>
  );
}
