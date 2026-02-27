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
      <p className="font-medium mb-1">Revocation complete:</p>
      <ul className="list-disc list-inside">
        <li>Sessions revoked: {result.revoked.sessions}</li>
        <li>Access tokens revoked: {result.revoked.access_tokens}</li>
        <li>Refresh tokens revoked: {result.revoked.refresh_tokens}</li>
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
        title="Incident Response"
        description="Emergency token revocation. These actions are irreversible."
      />

      {error && <Alert variant="error">{error}</Alert>}
      {result && <RevokeResult result={result} />}

      {/* Revoke All */}
      <Card title="Revoke All Tokens" variant="danger" className="mb-4">
        <p className="text-sm text-gray-600 mb-3">
          Revoke all sessions, access tokens, and refresh tokens across all
          tenants.
        </p>
        <div className="flex gap-2 items-end">
          <div className="flex-1">
            <label className="block text-xs text-gray-500 mb-1">
              Type &quot;REVOKE ALL&quot; to confirm
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
            Revoke All
          </button>
        </div>
      </Card>

      {/* Revoke by Tenant */}
      <Card title="Revoke by Tenant" className="mb-4">
        <p className="text-sm text-gray-600 mb-3">
          Revoke all tokens for a specific tenant.
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
                placeholder="Tenant ID (UUID)"
                className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </FormField>
          </div>
          <button
            type="submit"
            disabled={isLoading}
            className="px-4 py-2 bg-orange-500 text-white text-sm rounded hover:bg-orange-600 disabled:opacity-50 disabled:cursor-not-allowed self-start"
          >
            Revoke
          </button>
        </form>
      </Card>

      {/* Revoke by User */}
      <Card title="Revoke by User">
        <p className="text-sm text-gray-600 mb-3">
          Revoke all tokens for a specific user.
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
                placeholder="User ID (UUID)"
                className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </FormField>
          </div>
          <button
            type="submit"
            disabled={isLoading}
            className="px-4 py-2 bg-orange-500 text-white text-sm rounded hover:bg-orange-600 disabled:opacity-50 disabled:cursor-not-allowed self-start"
          >
            Revoke
          </button>
        </form>
      </Card>
    </div>
  );
}
