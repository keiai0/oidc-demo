"use client";

import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { tenantsApi } from "@/lib/api/tenants";
import { getErrorMessage } from "@/lib/fetcher";
import { createTenantSchema } from "@/schemas/tenant";
import type { CreateTenantInput } from "@/schemas/tenant";
import { routes } from "@/lib/routes";
import { Alert } from "@/components/ui/alert";
import { PageHeader } from "@/components/ui/page-header";
import { FormField } from "@/components/form/form-field";

export default function NewTenantPage() {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<CreateTenantInput>({
    resolver: zodResolver(createTenantSchema),
    defaultValues: {
      code: "",
      name: "",
      session_lifetime: 3600,
      auth_code_lifetime: 60,
      access_token_lifetime: 3600,
      refresh_token_lifetime: 2592000,
      id_token_lifetime: 3600,
    },
  });

  const mutation = useMutation({
    mutationFn: (data: CreateTenantInput) => tenantsApi.create(data),
    onSuccess: (tenant) => {
      window.location.href = routes.management.tenantDetail(tenant.id);
    },
  });

  const onSubmit = (data: CreateTenantInput) => {
    mutation.mutate(data);
  };

  return (
    <div className="max-w-lg">
      <PageHeader title="テナント作成" />

      {mutation.error && (
        <Alert variant="error">{getErrorMessage(mutation.error)}</Alert>
      )}

      <form
        onSubmit={handleSubmit(onSubmit)}
        className="bg-white rounded-lg border border-gray-200 p-6 space-y-4"
      >
        <FormField label="コード" error={errors.code}>
          <input
            {...register("code")}
            placeholder="my-tenant"
            className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </FormField>

        <FormField label="名前" error={errors.name}>
          <input
            {...register("name")}
            placeholder="My Tenant"
            className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </FormField>

        <FormField label="セッション有効期間 (秒)" error={errors.session_lifetime}>
          <input
            type="number"
            {...register("session_lifetime", { valueAsNumber: true })}
            className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </FormField>

        <FormField label="認可コード有効期間 (秒)" error={errors.auth_code_lifetime}>
          <input
            type="number"
            {...register("auth_code_lifetime", { valueAsNumber: true })}
            className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </FormField>

        <FormField label="アクセストークン有効期間 (秒)" error={errors.access_token_lifetime}>
          <input
            type="number"
            {...register("access_token_lifetime", { valueAsNumber: true })}
            className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </FormField>

        <FormField label="リフレッシュトークン有効期間 (秒)" error={errors.refresh_token_lifetime}>
          <input
            type="number"
            {...register("refresh_token_lifetime", { valueAsNumber: true })}
            className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </FormField>

        <FormField label="IDトークン有効期間 (秒)" error={errors.id_token_lifetime}>
          <input
            type="number"
            {...register("id_token_lifetime", { valueAsNumber: true })}
            className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </FormField>

        <div className="flex gap-3 pt-2">
          <button
            type="submit"
            disabled={mutation.isPending}
            className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700 disabled:opacity-50"
          >
            {mutation.isPending ? "作成中..." : "作成"}
          </button>
          <button
            type="button"
            onClick={() => history.back()}
            className="px-4 py-2 border border-gray-300 text-sm rounded text-gray-700 hover:bg-gray-50"
          >
            キャンセル
          </button>
        </div>
      </form>
    </div>
  );
}
