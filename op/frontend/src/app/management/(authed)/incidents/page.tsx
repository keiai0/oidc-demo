"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { incidentsApi } from "@/lib/api/incidents";
import { getErrorMessage } from "@/lib/fetcher";
import { revokeByTenantSchema, revokeByUserSchema } from "@/schemas/incident";
import type { RevokeByTenantInput, RevokeByUserInput } from "@/schemas/incident";
import type { RevokeResponse } from "@/types";
import { Alert } from "@/components/ui/alert";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { FormField } from "@/components/form/form-field";

function RevokeResult({ result }: { result: RevokeResponse }) {
  return (
    <Alert variant="success">
      <p className="font-medium mb-1">失効完了:</p>
      <ul className="list-disc list-inside">
        <li>セッション失効数: {result.revoked.sessions}</li>
        <li>アクセストークン失効数: {result.revoked.access_tokens}</li>
        <li>リフレッシュトークン失効数: {result.revoked.refresh_tokens}</li>
      </ul>
    </Alert>
  );
}

export default function IncidentsPage() {
  const [error, setError] = useState("");
  const [result, setResult] = useState<RevokeResponse | null>(null);
  const [revokeAllConfirm, setRevokeAllConfirm] = useState("");

  const revokeAllMutation = useMutation({
    mutationFn: incidentsApi.revokeAll,
    onSuccess: (data) => {
      setResult(data);
      setError("");
    },
    onError: (err) => {
      setError(getErrorMessage(err));
      setResult(null);
    },
  });

  const revokeByTenantMutation = useMutation({
    mutationFn: (data: RevokeByTenantInput) =>
      incidentsApi.revokeByTenant(data.tenant_id),
    onSuccess: (data) => {
      setResult(data);
      setError("");
    },
    onError: (err) => {
      setError(getErrorMessage(err));
      setResult(null);
    },
  });

  const revokeByUserMutation = useMutation({
    mutationFn: (data: RevokeByUserInput) =>
      incidentsApi.revokeByUser(data.user_id),
    onSuccess: (data) => {
      setResult(data);
      setError("");
    },
    onError: (err) => {
      setError(getErrorMessage(err));
      setResult(null);
    },
  });

  const tenantForm = useForm<RevokeByTenantInput>({
    resolver: zodResolver(revokeByTenantSchema),
    defaultValues: { tenant_id: "" },
  });

  const userForm = useForm<RevokeByUserInput>({
    resolver: zodResolver(revokeByUserSchema),
    defaultValues: { user_id: "" },
  });

  const isLoading =
    revokeAllMutation.isPending ||
    revokeByTenantMutation.isPending ||
    revokeByUserMutation.isPending;

  return (
    <div className="max-w-2xl">
      <PageHeader
        title="インシデント対応"
        description="緊急トークン失効。これらの操作は取り消せません。"
      />

      {error && <Alert variant="error">{error}</Alert>}
      {result && <RevokeResult result={result} />}

      {/* Revoke All */}
      <Card title="全トークン失効" variant="danger" className="mb-4">
        <p className="text-sm text-gray-600 mb-3">
          全テナントのセッション、アクセストークン、リフレッシュトークンをすべて失効させます。
        </p>
        <div className="flex gap-2 items-end">
          <div className="flex-1">
            <label className="block text-xs text-gray-500 mb-1">
              確認のため「REVOKE ALL」と入力してください
            </label>
            <input
              value={revokeAllConfirm}
              onChange={(e) => setRevokeAllConfirm(e.target.value)}
              placeholder="REVOKE ALL"
              className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-red-500"
            />
          </div>
          <button
            onClick={() => revokeAllMutation.mutate()}
            disabled={revokeAllConfirm !== "REVOKE ALL" || isLoading}
            className="px-4 py-2 bg-red-600 text-white text-sm rounded hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            全失効
          </button>
        </div>
      </Card>

      {/* Revoke by Tenant */}
      <Card title="テナント単位で失効" className="mb-4">
        <p className="text-sm text-gray-600 mb-3">
          指定したテナントの全トークンを失効させます。
        </p>
        <form
          onSubmit={tenantForm.handleSubmit((data) =>
            revokeByTenantMutation.mutate(data),
          )}
          className="flex gap-2"
        >
          <div className="flex-1">
            <FormField label="" error={tenantForm.formState.errors.tenant_id}>
              <input
                {...tenantForm.register("tenant_id")}
                placeholder="テナントID (UUID)"
                className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </FormField>
          </div>
          <button
            type="submit"
            disabled={isLoading}
            className="px-4 py-2 bg-orange-500 text-white text-sm rounded hover:bg-orange-600 disabled:opacity-50 disabled:cursor-not-allowed self-start"
          >
            失効
          </button>
        </form>
      </Card>

      {/* Revoke by User */}
      <Card title="ユーザー単位で失効">
        <p className="text-sm text-gray-600 mb-3">
          指定したユーザーの全トークンを失効させます。
        </p>
        <form
          onSubmit={userForm.handleSubmit((data) =>
            revokeByUserMutation.mutate(data),
          )}
          className="flex gap-2"
        >
          <div className="flex-1">
            <FormField label="" error={userForm.formState.errors.user_id}>
              <input
                {...userForm.register("user_id")}
                placeholder="ユーザーID (UUID)"
                className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </FormField>
          </div>
          <button
            type="submit"
            disabled={isLoading}
            className="px-4 py-2 bg-orange-500 text-white text-sm rounded hover:bg-orange-600 disabled:opacity-50 disabled:cursor-not-allowed self-start"
          >
            失効
          </button>
        </form>
      </Card>
    </div>
  );
}
