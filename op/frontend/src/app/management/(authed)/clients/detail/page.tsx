"use client";

import Link from "next/link";
import { Fragment, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { clientsApi } from "@/lib/api/clients";
import { tokenExchangePoliciesApi } from "@/lib/api/token-exchange-policies";
import { tenantsApi } from "@/lib/api/tenants";
import { getErrorMessage } from "@/lib/fetcher";
import { queryKeys } from "@/lib/query/query-keys";
import { routes } from "@/lib/routes";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Loading } from "@/components/ui/loading";

export default function ClientDetailPage() {
  const searchParams = useSearchParams();
  const id = searchParams.get("id") ?? "";
  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const [newRedirectURI, setNewRedirectURI] = useState("");
  const [newPostLogoutURI, setNewPostLogoutURI] = useState("");
  const [newSecret, setNewSecret] = useState<string | null>(null);
  const [selectedTenantId, setSelectedTenantId] = useState("");
  const [editingOidc, setEditingOidc] = useState(false);
  const [subjectType, setSubjectType] = useState("");
  const [sectorIdentifierUri, setSectorIdentifierUri] = useState("");
  const [userinfoSignedAlg, setUserinfoSignedAlg] = useState("");
  const [editingTokenExchange, setEditingTokenExchange] = useState(false);
  const [teAllowImpersonation, setTeAllowImpersonation] = useState(false);
  const [teAllowDelegation, setTeAllowDelegation] = useState(true);

  const { data: client, isLoading } = useQuery({
    queryKey: queryKeys.clients.detail(id),
    queryFn: () => clientsApi.get(id),
    enabled: !!id,
  });

  const { data: tenantAssociations } = useQuery({
    queryKey: queryKeys.clients.tenants(id),
    queryFn: () => clientsApi.listTenants(id),
    enabled: !!id,
  });

  const { data: allTenants } = useQuery({
    queryKey: queryKeys.tenants.list(250, 0),
    queryFn: () => tenantsApi.list(250, 0),
  });

  const { data: tokenExchangePolicy } = useQuery({
    queryKey: queryKeys.clients.tokenExchangePolicy(id),
    queryFn: () => tokenExchangePoliciesApi.get(id).catch(() => null),
    enabled: !!id,
  });

  const invalidateClient = () =>
    queryClient.invalidateQueries({ queryKey: queryKeys.clients.detail(id) });

  const invalidateTokenExchangePolicy = () =>
    queryClient.invalidateQueries({
      queryKey: queryKeys.clients.tokenExchangePolicy(id),
    });

  const invalidateTenants = () =>
    queryClient.invalidateQueries({ queryKey: queryKeys.clients.tenants(id) });

  const addRedirectURIMutation = useMutation({
    mutationFn: (uri: string) => clientsApi.addRedirectURI(id, uri),
    onSuccess: () => {
      setNewRedirectURI("");
      setError("");
      invalidateClient();
    },
    onError: (err) => setError(getErrorMessage(err)),
  });

  const deleteRedirectURIMutation = useMutation({
    mutationFn: (uriId: string) => clientsApi.deleteRedirectURI(id, uriId),
    onSuccess: () => {
      setError("");
      invalidateClient();
    },
    onError: (err) => setError(getErrorMessage(err)),
  });

  const addPostLogoutURIMutation = useMutation({
    mutationFn: (uri: string) => clientsApi.addPostLogoutRedirectURI(id, uri),
    onSuccess: () => {
      setNewPostLogoutURI("");
      setError("");
      invalidateClient();
    },
    onError: (err) => setError(getErrorMessage(err)),
  });

  const deletePostLogoutURIMutation = useMutation({
    mutationFn: (uriId: string) =>
      clientsApi.deletePostLogoutRedirectURI(id, uriId),
    onSuccess: () => {
      setError("");
      invalidateClient();
    },
    onError: (err) => setError(getErrorMessage(err)),
  });

  const rotateSecretMutation = useMutation({
    mutationFn: () => clientsApi.rotateSecret(id),
    onSuccess: (data) => {
      setNewSecret(data.client_secret);
      setError("");
    },
    onError: (err) => setError(getErrorMessage(err)),
  });

  const deleteMutation = useMutation({
    mutationFn: () => clientsApi.delete(id),
    onSuccess: () => {
      window.location.href = routes.management.clients;
    },
    onError: (err) => setError(getErrorMessage(err)),
  });

  const addTenantMutation = useMutation({
    mutationFn: (tenantId: string) => clientsApi.addTenant(id, tenantId),
    onSuccess: () => {
      setSelectedTenantId("");
      setError("");
      invalidateTenants();
    },
    onError: (err) => setError(getErrorMessage(err)),
  });

  const removeTenantMutation = useMutation({
    mutationFn: (tenantId: string) => clientsApi.removeTenant(id, tenantId),
    onSuccess: () => {
      setError("");
      invalidateTenants();
    },
    onError: (err) => setError(getErrorMessage(err)),
  });

  const saveTokenExchangePolicyMutation = useMutation({
    mutationFn: () =>
      tokenExchangePoliciesApi.createOrUpdate(id, {
        allowed_subject_token_types: [
          "urn:ietf:params:oauth:token-type:access_token",
        ],
        allowed_requested_token_types: [
          "urn:ietf:params:oauth:token-type:access_token",
        ],
        allowed_audiences: [],
        allow_impersonation: teAllowImpersonation,
        allow_delegation: teAllowDelegation,
      }),
    onSuccess: () => {
      setError("");
      setEditingTokenExchange(false);
      invalidateTokenExchangePolicy();
    },
    onError: (err) => setError(getErrorMessage(err)),
  });

  const deleteTokenExchangePolicyMutation = useMutation({
    mutationFn: () => tokenExchangePoliciesApi.delete(id),
    onSuccess: () => {
      setError("");
      invalidateTokenExchangePolicy();
    },
    onError: (err) => setError(getErrorMessage(err)),
  });

  const startEditTokenExchange = () => {
    if (tokenExchangePolicy) {
      setTeAllowImpersonation(tokenExchangePolicy.allow_impersonation);
      setTeAllowDelegation(tokenExchangePolicy.allow_delegation);
    } else {
      setTeAllowImpersonation(false);
      setTeAllowDelegation(true);
    }
    setEditingTokenExchange(true);
  };

  const updateOidcMutation = useMutation({
    mutationFn: () =>
      clientsApi.update(id, {
        subject_type: subjectType,
        sector_identifier_uri: sectorIdentifierUri || undefined,
        userinfo_signed_response_alg: userinfoSignedAlg || undefined,
      }),
    onSuccess: () => {
      setError("");
      setEditingOidc(false);
      invalidateClient();
    },
    onError: (err) => setError(getErrorMessage(err)),
  });

  const startEditOidc = () => {
    if (client) {
      setSubjectType(client.subject_type ?? "public");
      setSectorIdentifierUri(client.sector_identifier_uri ?? "");
      setUserinfoSignedAlg(client.userinfo_signed_response_alg ?? "");
    }
    setEditingOidc(true);
  };

  const handleRotateSecret = () => {
    if (!confirm("現在のシークレットは無効になります。よろしいですか？"))
      return;
    rotateSecretMutation.mutate();
  };

  const handleDelete = () => {
    if (!confirm("このクライアントを削除しますか？")) return;
    deleteMutation.mutate();
  };

  if (isLoading) return <Loading />;
  if (!client) return <p className="text-gray-500">クライアントが見つかりません</p>;

  const associatedTenantIds = new Set(
    tenantAssociations?.map((ta) => ta.tenant_id) ?? [],
  );
  const availableTenants =
    allTenants?.data.filter((t) => !associatedTenantIds.has(t.id)) ?? [];

  const infoFields = [
    ["Client ID", client.client_id],
    ["認証方式", client.token_endpoint_auth_method],
    ["Grant Types", client.grant_types.join(", ")],
    ["Response Types", client.response_types.join(", ")],
    ["PKCE", client.require_pkce ? "必須" : "任意"],
    ["作成日時", new Date(client.created_at).toLocaleString()],
    ["更新日時", new Date(client.updated_at).toLocaleString()],
  ] as const;

  return (
    <div className="max-w-2xl">
      <Link
        href={routes.management.clients}
        className="text-sm text-blue-600 hover:underline"
      >
        &larr; クライアント一覧に戻る
      </Link>
      <div className="flex items-center justify-between mt-1 mb-6">
        <h1 className="text-2xl font-bold text-gray-900">{client.name}</h1>
        <Badge variant={client.status === "active" ? "active" : "inactive"}>
          {client.status}
        </Badge>
      </div>

      {error && <Alert variant="error">{error}</Alert>}

      {/* Client Info */}
      <Card title="クライアント情報" className="mb-4">
        <dl className="grid grid-cols-2 gap-x-6 gap-y-3 text-sm">
          {infoFields.map(([label, value]) => (
            <Fragment key={label}>
              <dt className="text-gray-500">{label}</dt>
              <dd className={label === "Client ID" ? "font-mono text-xs" : ""}>
                {value}
              </dd>
            </Fragment>
          ))}
        </dl>
      </Card>

      {/* OIDC Settings */}
      <Card
        title="OIDC 設定"
        titleAction={
          !editingOidc ? (
            <button
              onClick={startEditOidc}
              className="text-sm text-blue-600 hover:underline"
            >
              編集
            </button>
          ) : undefined
        }
        className="mb-4"
      >
        {editingOidc ? (
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Subject Type (OIDC Core Section 8)
              </label>
              <select
                value={subjectType}
                onChange={(e) => setSubjectType(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="public">public（デフォルト）</option>
                <option value="pairwise">pairwise（RP ごとに異なる sub）</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Sector Identifier URI
              </label>
              <input
                value={sectorIdentifierUri}
                onChange={(e) => setSectorIdentifierUri(e.target.value)}
                placeholder="https://example.com（空の場合は redirect_uri のホストを使用）"
                className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
              <p className="text-xs text-gray-500 mt-1">
                pairwise sub の計算に使用する sector 識別子。空の場合はリダイレクト URI のホスト部を使用します。
              </p>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Userinfo 署名アルゴリズム (OIDC Core Section 5.3.2)
              </label>
              <select
                value={userinfoSignedAlg}
                onChange={(e) => setUserinfoSignedAlg(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">なし（JSON レスポンス）</option>
                <option value="RS256">RS256（署名付き JWT レスポンス）</option>
              </select>
            </div>
            <div className="flex gap-3 pt-2">
              <button
                onClick={() => updateOidcMutation.mutate()}
                disabled={updateOidcMutation.isPending}
                className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700 disabled:opacity-50"
              >
                {updateOidcMutation.isPending ? "保存中..." : "保存"}
              </button>
              <button
                onClick={() => setEditingOidc(false)}
                className="px-4 py-2 border border-gray-300 text-sm rounded text-gray-700 hover:bg-gray-50"
              >
                キャンセル
              </button>
            </div>
          </div>
        ) : (
          <dl className="grid grid-cols-2 gap-x-6 gap-y-3 text-sm">
            <dt className="text-gray-500">Subject Type</dt>
            <dd>{client.subject_type ?? "public"}</dd>
            <dt className="text-gray-500">Sector Identifier URI</dt>
            <dd className="font-mono text-xs">{client.sector_identifier_uri || "（未設定）"}</dd>
            <dt className="text-gray-500">Userinfo 署名</dt>
            <dd>{client.userinfo_signed_response_alg || "なし（JSON）"}</dd>
          </dl>
        )}
      </Card>

      {/* Tenant Associations */}
      <Card title="関連テナント" className="mb-4">
        {!tenantAssociations || tenantAssociations.length === 0 ? (
          <p className="text-sm text-gray-500 mb-3">
            関連テナントがありません
          </p>
        ) : (
          <ul className="space-y-2 mb-3">
            {tenantAssociations.map((ta) => (
              <li
                key={ta.tenant_id}
                className="flex items-center justify-between bg-gray-50 px-3 py-2 rounded text-sm"
              >
                <Link
                  href={routes.management.tenantDetail(ta.tenant_id)}
                  className="text-blue-600 hover:underline"
                >
                  {ta.tenant_name}{" "}
                  <span className="text-gray-400 text-xs">
                    ({ta.tenant_code})
                  </span>
                </Link>
                <button
                  onClick={() => removeTenantMutation.mutate(ta.tenant_id)}
                  className="text-red-500 hover:text-red-700 text-xs ml-2 shrink-0"
                >
                  解除
                </button>
              </li>
            ))}
          </ul>
        )}
        {availableTenants.length > 0 && (
          <div className="flex gap-2">
            <select
              value={selectedTenantId}
              onChange={(e) => setSelectedTenantId(e.target.value)}
              className="flex-1 px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="">テナントを選択...</option>
              {availableTenants.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name} ({t.code})
                </option>
              ))}
            </select>
            <button
              onClick={() => addTenantMutation.mutate(selectedTenantId)}
              disabled={!selectedTenantId}
              className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700 disabled:opacity-50"
            >
              追加
            </button>
          </div>
        )}
      </Card>

      {/* Secret Rotation */}
      <Card title="クライアントシークレット" className="mb-4">
        {newSecret && (
          <Alert variant="warning">
            <p className="font-medium mb-1">新しいシークレット（一度しか表示されません）:</p>
            <code className="block bg-white px-3 py-2 rounded text-sm font-mono break-all border">
              {newSecret}
            </code>
          </Alert>
        )}
        <button
          onClick={handleRotateSecret}
          className="px-4 py-2 bg-yellow-500 text-white text-sm rounded hover:bg-yellow-600"
        >
          シークレット再生成
        </button>
      </Card>

      {/* Redirect URIs */}
      <Card title="リダイレクト URI" className="mb-4">
        {client.redirect_uris.length === 0 ? (
          <p className="text-sm text-gray-500 mb-3">リダイレクト URI がありません</p>
        ) : (
          <ul className="space-y-2 mb-3">
            {client.redirect_uris.map((ru) => (
              <li
                key={ru.id}
                className="flex items-center justify-between bg-gray-50 px-3 py-2 rounded text-sm"
              >
                <span className="font-mono text-xs break-all">{ru.uri}</span>
                <button
                  onClick={() => deleteRedirectURIMutation.mutate(ru.id)}
                  className="text-red-500 hover:text-red-700 text-xs ml-2 shrink-0"
                >
                  削除
                </button>
              </li>
            ))}
          </ul>
        )}
        <div className="flex gap-2">
          <input
            value={newRedirectURI}
            onChange={(e) => setNewRedirectURI(e.target.value)}
            placeholder="https://example.com/callback"
            className="flex-1 px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
          <button
            onClick={() => addRedirectURIMutation.mutate(newRedirectURI.trim())}
            disabled={!newRedirectURI.trim()}
            className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700 disabled:opacity-50"
          >
            追加
          </button>
        </div>
      </Card>

      {/* Post-Logout Redirect URIs */}
      <Card title="ログアウト後リダイレクト URI" className="mb-4">
        {client.post_logout_redirect_uris.length === 0 ? (
          <p className="text-sm text-gray-500 mb-3">
            ログアウト後リダイレクト URI がありません
          </p>
        ) : (
          <ul className="space-y-2 mb-3">
            {client.post_logout_redirect_uris.map((ru) => (
              <li
                key={ru.id}
                className="flex items-center justify-between bg-gray-50 px-3 py-2 rounded text-sm"
              >
                <span className="font-mono text-xs break-all">{ru.uri}</span>
                <button
                  onClick={() => deletePostLogoutURIMutation.mutate(ru.id)}
                  className="text-red-500 hover:text-red-700 text-xs ml-2 shrink-0"
                >
                  削除
                </button>
              </li>
            ))}
          </ul>
        )}
        <div className="flex gap-2">
          <input
            value={newPostLogoutURI}
            onChange={(e) => setNewPostLogoutURI(e.target.value)}
            placeholder="https://example.com/logout-callback"
            className="flex-1 px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
          <button
            onClick={() =>
              addPostLogoutURIMutation.mutate(newPostLogoutURI.trim())
            }
            disabled={!newPostLogoutURI.trim()}
            className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700 disabled:opacity-50"
          >
            追加
          </button>
        </div>
      </Card>

      {/* Token Exchange Policy (RFC 8693) */}
      <Card
        title="Token Exchange ポリシー (RFC 8693)"
        titleAction={
          !editingTokenExchange ? (
            <button
              onClick={startEditTokenExchange}
              className="text-sm text-blue-600 hover:underline"
            >
              {tokenExchangePolicy ? "編集" : "作成"}
            </button>
          ) : undefined
        }
        className="mb-4"
      >
        {editingTokenExchange ? (
          <div className="space-y-4">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={teAllowImpersonation}
                onChange={(e) => setTeAllowImpersonation(e.target.checked)}
                className="rounded border-gray-300"
              />
              Impersonation を許可（actor_token なしでのトークン交換）
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={teAllowDelegation}
                onChange={(e) => setTeAllowDelegation(e.target.checked)}
                className="rounded border-gray-300"
              />
              Delegation を許可（act クレーム付きトークン発行）
            </label>
            <div className="flex gap-3 pt-2">
              <button
                onClick={() => saveTokenExchangePolicyMutation.mutate()}
                disabled={saveTokenExchangePolicyMutation.isPending}
                className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700 disabled:opacity-50"
              >
                {saveTokenExchangePolicyMutation.isPending ? "保存中..." : "保存"}
              </button>
              <button
                onClick={() => setEditingTokenExchange(false)}
                className="px-4 py-2 border border-gray-300 text-sm rounded text-gray-700 hover:bg-gray-50"
              >
                キャンセル
              </button>
            </div>
          </div>
        ) : tokenExchangePolicy ? (
          <div>
            <dl className="grid grid-cols-2 gap-x-6 gap-y-3 text-sm">
              <dt className="text-gray-500">Impersonation</dt>
              <dd>{tokenExchangePolicy.allow_impersonation ? "許可" : "不許可"}</dd>
              <dt className="text-gray-500">Delegation</dt>
              <dd>{tokenExchangePolicy.allow_delegation ? "許可" : "不許可"}</dd>
            </dl>
            <button
              onClick={() => {
                if (confirm("Token Exchange ポリシーを削除しますか？")) {
                  deleteTokenExchangePolicyMutation.mutate();
                }
              }}
              className="mt-4 text-sm text-red-500 hover:text-red-700"
            >
              ポリシーを削除
            </button>
          </div>
        ) : (
          <p className="text-sm text-gray-500">
            ポリシーが設定されていません。作成すると Token Exchange が有効になります。
          </p>
        )}
      </Card>

      {/* Danger Zone */}
      <Card title="危険な操作" variant="danger">
        <button
          onClick={handleDelete}
          className="px-4 py-2 bg-red-600 text-white text-sm rounded hover:bg-red-700"
        >
          クライアント削除
        </button>
      </Card>
    </div>
  );
}
