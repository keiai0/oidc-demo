"use client";

import Link from "next/link";
import { Fragment, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { tenantsApi } from "@/lib/api/tenants";
import { getErrorMessage } from "@/lib/fetcher";
import { queryKeys } from "@/lib/query/query-keys";
import { routes } from "@/lib/routes";
import { updateTenantSchema } from "@/schemas/tenant";
import type { UpdateTenantInput } from "@/schemas/tenant";
import { Alert } from "@/components/ui/alert";
import { Card } from "@/components/ui/card";
import { Loading } from "@/components/ui/loading";
import { PageHeader } from "@/components/ui/page-header";

export default function TenantDetailPage() {
  const searchParams = useSearchParams();
  const id = searchParams.get("id") ?? "";
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState(false);
  const [success, setSuccess] = useState("");

  const { data: tenant, isLoading, error } = useQuery({
    queryKey: queryKeys.tenants.detail(id),
    queryFn: () => tenantsApi.get(id),
    enabled: !!id,
  });

  const form = useForm<UpdateTenantInput>({
    resolver: zodResolver(updateTenantSchema),
    values: tenant
      ? {
          name: tenant.name,
          session_lifetime: tenant.session_lifetime,
          auth_code_lifetime: tenant.auth_code_lifetime,
          access_token_lifetime: tenant.access_token_lifetime,
          refresh_token_lifetime: tenant.refresh_token_lifetime,
          id_token_lifetime: tenant.id_token_lifetime,
        }
      : undefined,
  });

  const updateMutation = useMutation({
    mutationFn: (data: UpdateTenantInput) => tenantsApi.update(id, data),
    onSuccess: () => {
      setEditing(false);
      setSuccess("Tenant updated successfully");
      queryClient.invalidateQueries({ queryKey: queryKeys.tenants.detail(id) });
    },
  });

  if (isLoading) return <Loading />;
  if (error) return <Alert variant="error">{getErrorMessage(error)}</Alert>;
  if (!tenant) return <p className="text-gray-500">Tenant not found.</p>;

  const lifetimeFields = [
    ["session_lifetime", "Session Lifetime"],
    ["auth_code_lifetime", "Auth Code Lifetime"],
    ["access_token_lifetime", "Access Token Lifetime"],
    ["refresh_token_lifetime", "Refresh Token Lifetime"],
    ["id_token_lifetime", "ID Token Lifetime"],
  ] as const;

  return (
    <div className="max-w-2xl">
      <PageHeader title={tenant.name} />

      {updateMutation.error && (
        <Alert variant="error">{getErrorMessage(updateMutation.error)}</Alert>
      )}
      {success && <Alert variant="success">{success}</Alert>}

      <Card
        title="Details"
        titleAction={
          !editing ? (
            <button
              onClick={() => setEditing(true)}
              className="text-sm text-blue-600 hover:underline"
            >
              Edit
            </button>
          ) : undefined
        }
        className="mb-6"
      >
        <dl className="grid grid-cols-2 gap-x-6 gap-y-3 text-sm">
          <dt className="text-gray-500">Code</dt>
          <dd className="font-mono">{tenant.code}</dd>

          <dt className="text-gray-500">Name</dt>
          <dd>
            {editing ? (
              <input
                {...form.register("name")}
                className="w-full px-2 py-1 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            ) : (
              tenant.name
            )}
          </dd>

          {lifetimeFields.map(([key, label]) => (
            <Fragment key={key}>
              <dt className="text-gray-500">{label}</dt>
              <dd>
                {editing ? (
                  <input
                    type="number"
                    {...form.register(key, { valueAsNumber: true })}
                    className="w-24 px-2 py-1 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                ) : (
                  `${tenant[key]}s`
                )}
              </dd>
            </Fragment>
          ))}

          <dt className="text-gray-500">Created</dt>
          <dd>{new Date(tenant.created_at).toLocaleString()}</dd>

          <dt className="text-gray-500">Updated</dt>
          <dd>{new Date(tenant.updated_at).toLocaleString()}</dd>
        </dl>

        {editing && (
          <div className="flex gap-3 mt-4 pt-4 border-t border-gray-200">
            <button
              onClick={form.handleSubmit((data) => updateMutation.mutate(data))}
              className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
            >
              Save
            </button>
            <button
              onClick={() => setEditing(false)}
              className="px-4 py-2 border border-gray-300 text-sm rounded text-gray-700 hover:bg-gray-50"
            >
              Cancel
            </button>
          </div>
        )}
      </Card>

      <Card title="Clients">
        <Link
          href={`${routes.management.tenantClients(id)}`}
          className="text-sm text-blue-600 hover:underline"
        >
          View All
        </Link>
      </Card>
    </div>
  );
}
