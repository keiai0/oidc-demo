"use client";

import { useState } from "react";
import { useSearchParams } from "next/navigation";
import { useMutation } from "@tanstack/react-query";
import { federationProvidersApi } from "@/lib/api/federation-providers";
import { getErrorMessage } from "@/lib/fetcher";
import { routes } from "@/lib/routes";
import { Alert } from "@/components/ui/alert";
import { PageHeader } from "@/components/ui/page-header";

export default function NewFederationProviderPage() {
  const searchParams = useSearchParams();
  const tenantId = searchParams.get("tenant_id") || "";

  const [name, setName] = useState("");
  const [issuer, setIssuer] = useState("");
  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [scopes, setScopes] = useState("openid profile email");
  const [autoProvision, setAutoProvision] = useState(true);
  const [error, setError] = useState("");

  const mutation = useMutation({
    mutationFn: () =>
      federationProvidersApi.create(tenantId, {
        name,
        issuer,
        client_id: clientId,
        client_secret: clientSecret,
        scopes,
        auto_provision: autoProvision,
      }),
    onSuccess: () => {
      window.location.href = routes.management.federationProviders;
    },
    onError: (err) => {
      setError(getErrorMessage(err));
    },
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    mutation.mutate();
  }

  const inputClass =
    "w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent";

  return (
    <div>
      <PageHeader title="IdP 連携プロバイダ作成" />

      {error && (
        <Alert variant="error" className="mb-4">
          {error}
        </Alert>
      )}

      <form
        onSubmit={handleSubmit}
        className="bg-white border border-gray-200 rounded-lg p-6 max-w-lg space-y-4"
      >
        <div>
          <label className="block text-sm font-medium text-gray-600 mb-1">
            名前 (URL パスに使用)
          </label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="google"
            required
            className={inputClass}
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-600 mb-1">
            Issuer URL
          </label>
          <input
            type="url"
            value={issuer}
            onChange={(e) => setIssuer(e.target.value)}
            placeholder="https://accounts.google.com"
            required
            className={inputClass}
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-600 mb-1">
            Client ID
          </label>
          <input
            type="text"
            value={clientId}
            onChange={(e) => setClientId(e.target.value)}
            required
            className={inputClass}
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-600 mb-1">
            Client Secret
          </label>
          <input
            type="password"
            value={clientSecret}
            onChange={(e) => setClientSecret(e.target.value)}
            required
            className={inputClass}
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-600 mb-1">
            Scopes
          </label>
          <input
            type="text"
            value={scopes}
            onChange={(e) => setScopes(e.target.value)}
            className={inputClass}
          />
        </div>

        <div className="flex items-center gap-2">
          <input
            type="checkbox"
            id="autoProvision"
            checked={autoProvision}
            onChange={(e) => setAutoProvision(e.target.checked)}
            className="rounded"
          />
          <label htmlFor="autoProvision" className="text-sm text-gray-600">
            JIT プロビジョニング (初回ログイン時にユーザーを自動作成)
          </label>
        </div>

        <div className="flex gap-2 pt-2">
          <button
            type="button"
            onClick={() =>
              (window.location.href = routes.management.federationProviders)
            }
            className="px-4 py-2 border border-gray-300 rounded text-sm hover:bg-gray-50"
          >
            キャンセル
          </button>
          <button
            type="submit"
            disabled={mutation.isPending}
            className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700 disabled:opacity-50"
          >
            {mutation.isPending ? "作成中..." : "作成"}
          </button>
        </div>
      </form>
    </div>
  );
}
