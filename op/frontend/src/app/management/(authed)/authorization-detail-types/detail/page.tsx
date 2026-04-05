"use client";

import { useState } from "react";
import { useSearchParams } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { authorizationDetailTypesApi } from "@/lib/api/authorization-detail-types";
import { getErrorMessage } from "@/lib/fetcher";
import { queryKeys } from "@/lib/query/query-keys";
import { routes } from "@/lib/routes";
import { Alert } from "@/components/ui/alert";
import { Loading } from "@/components/ui/loading";
import { PageHeader } from "@/components/ui/page-header";

export default function AuthorizationDetailTypeDetailPage() {
  const searchParams = useSearchParams();
  const id = searchParams.get("id") || "";
  const isNew = searchParams.get("new") === "true";
  const tenantId = searchParams.get("tenant_id") || "";
  const queryClient = useQueryClient();

  const [editing, setEditing] = useState(isNew);
  const [typeName, setTypeName] = useState("");
  const [description, setDescription] = useState("");
  const [allowedActions, setAllowedActions] = useState("");
  const [allowedLocations, setAllowedLocations] = useState("");
  const [jsonSchema, setJsonSchema] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const { data: detail, isLoading } = useQuery({
    queryKey: queryKeys.authorizationDetailTypes.detail(id),
    queryFn: () => authorizationDetailTypesApi.get(id),
    enabled: !!id && !isNew,
  });

  function startEdit() {
    if (!detail) return;
    setTypeName(detail.type_name);
    setDescription(detail.description);
    setAllowedActions(detail.allowed_actions.join(", "));
    setAllowedLocations(detail.allowed_locations.join(", "));
    setJsonSchema(detail.json_schema ?? "");
    setEditing(true);
    setError("");
    setSuccess("");
  }

  function parseCommaSeparated(value: string): string[] {
    return value
      .split(",")
      .map((s) => s.trim())
      .filter((s) => s.length > 0);
  }

  const createMutation = useMutation({
    mutationFn: () =>
      authorizationDetailTypesApi.create(tenantId, {
        type_name: typeName,
        description,
        json_schema: jsonSchema || undefined,
        allowed_actions: parseCommaSeparated(allowedActions),
        allowed_locations: parseCommaSeparated(allowedLocations),
      }),
    onSuccess: () => {
      window.location.href = routes.management.authorizationDetailTypes;
    },
    onError: (err) => setError(getErrorMessage(err)),
  });

  const updateMutation = useMutation({
    mutationFn: () =>
      authorizationDetailTypesApi.update(id, {
        type_name: typeName,
        description,
        json_schema: jsonSchema || null,
        allowed_actions: parseCommaSeparated(allowedActions),
        allowed_locations: parseCommaSeparated(allowedLocations),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.authorizationDetailTypes.detail(id),
      });
      setEditing(false);
      setSuccess("更新しました");
    },
    onError: (err) => setError(getErrorMessage(err)),
  });

  const deleteMutation = useMutation({
    mutationFn: () => authorizationDetailTypesApi.delete(id),
    onSuccess: () => {
      window.location.href = routes.management.authorizationDetailTypes;
    },
    onError: (err) => setError(getErrorMessage(err)),
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    if (isNew) {
      createMutation.mutate();
    } else {
      updateMutation.mutate();
    }
  }

  if (!isNew && isLoading) return <Loading />;
  if (!isNew && !detail)
    return <Alert variant="error">認可詳細タイプが見つかりません</Alert>;

  const isPending = createMutation.isPending || updateMutation.isPending;

  const inputClass =
    "w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent";

  return (
    <div>
      <PageHeader
        title={isNew ? "認可詳細タイプ作成" : `認可詳細タイプ: ${detail!.type_name}`}
        action={
          !isNew && !editing ? (
            <div className="flex gap-2">
              <button
                onClick={startEdit}
                className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
              >
                編集
              </button>
              <button
                onClick={() => {
                  if (confirm("この認可詳細タイプを削除しますか？")) {
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

      {error && (
        <Alert variant="error" className="mb-4">
          {error}
        </Alert>
      )}
      {success && (
        <Alert variant="success" className="mb-4">
          {success}
        </Alert>
      )}

      <div className="bg-white border border-gray-200 rounded-lg p-6 max-w-lg space-y-4">
        {editing ? (
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1">
                タイプ名
              </label>
              <input
                type="text"
                value={typeName}
                onChange={(e) => setTypeName(e.target.value)}
                placeholder="payment_initiation"
                required
                className={inputClass}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1">
                説明
              </label>
              <input
                type="text"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="決済開始の認可詳細タイプ"
                className={inputClass}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1">
                許可アクション (カンマ区切り)
              </label>
              <input
                type="text"
                value={allowedActions}
                onChange={(e) => setAllowedActions(e.target.value)}
                placeholder="read, write, execute"
                className={inputClass}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1">
                許可ロケーション (カンマ区切り)
              </label>
              <input
                type="text"
                value={allowedLocations}
                onChange={(e) => setAllowedLocations(e.target.value)}
                placeholder="https://api.example.com/accounts"
                className={inputClass}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-600 mb-1">
                JSON Schema (任意)
              </label>
              <textarea
                value={jsonSchema}
                onChange={(e) => setJsonSchema(e.target.value)}
                placeholder='{"type": "object", "properties": {...}}'
                rows={6}
                className={inputClass}
              />
            </div>
            <div className="flex gap-2 pt-2">
              <button
                type="button"
                onClick={() => {
                  if (isNew) {
                    window.location.href =
                      routes.management.authorizationDetailTypes;
                  } else {
                    setEditing(false);
                  }
                }}
                className="px-4 py-2 border border-gray-300 rounded text-sm hover:bg-gray-50"
              >
                キャンセル
              </button>
              <button
                type="submit"
                disabled={isPending}
                className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700 disabled:opacity-50"
              >
                {isPending
                  ? isNew
                    ? "作成中..."
                    : "保存中..."
                  : isNew
                    ? "作成"
                    : "保存"}
              </button>
            </div>
          </form>
        ) : (
          <dl className="space-y-3">
            <div>
              <dt className="text-xs text-gray-500">タイプ名</dt>
              <dd className="font-medium">{detail!.type_name}</dd>
            </div>
            <div>
              <dt className="text-xs text-gray-500">説明</dt>
              <dd className="text-sm">{detail!.description || "-"}</dd>
            </div>
            <div>
              <dt className="text-xs text-gray-500">許可アクション</dt>
              <dd className="text-sm">
                {detail!.allowed_actions.length > 0
                  ? detail!.allowed_actions.join(", ")
                  : "-"}
              </dd>
            </div>
            <div>
              <dt className="text-xs text-gray-500">許可ロケーション</dt>
              <dd className="text-sm font-mono break-all">
                {detail!.allowed_locations.length > 0
                  ? detail!.allowed_locations.join(", ")
                  : "-"}
              </dd>
            </div>
            <div>
              <dt className="text-xs text-gray-500">JSON Schema</dt>
              <dd className="text-sm">
                {detail!.json_schema ? (
                  <pre className="bg-gray-50 p-2 rounded text-xs font-mono overflow-auto max-h-48">
                    {detail!.json_schema}
                  </pre>
                ) : (
                  "-"
                )}
              </dd>
            </div>
            <div>
              <dt className="text-xs text-gray-500">作成日時</dt>
              <dd className="text-sm text-gray-600">
                {new Date(detail!.created_at).toLocaleString()}
              </dd>
            </div>
            <div>
              <dt className="text-xs text-gray-500">更新日時</dt>
              <dd className="text-sm text-gray-600">
                {new Date(detail!.updated_at).toLocaleString()}
              </dd>
            </div>
          </dl>
        )}
      </div>
    </div>
  );
}
