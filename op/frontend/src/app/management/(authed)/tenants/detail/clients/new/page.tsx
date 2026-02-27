"use client";

import Link from "next/link";
import { useState } from "react";
import { useSearchParams } from "next/navigation";
import { useMutation } from "@tanstack/react-query";
import { clientsApi } from "@/lib/api/clients";
import { getErrorMessage } from "@/lib/fetcher";
import { routes } from "@/lib/routes";
import { Alert } from "@/components/ui/alert";
import { PageHeader } from "@/components/ui/page-header";
import { URIListField } from "@/components/form/uri-list-field";
import type { ClientCreateResponse } from "@/types";

export default function NewClientPage() {
  const searchParams = useSearchParams();
  const tenantId = searchParams.get("tenant_id") ?? "";

  const [name, setName] = useState("");
  const [authMethod, setAuthMethod] = useState("client_secret_basic");
  const [requirePkce, setRequirePkce] = useState(true);
  const [redirectURIs, setRedirectURIs] = useState<string[]>([""]);
  const [postLogoutRedirectURIs, setPostLogoutRedirectURIs] = useState<string[]>([""]);
  const [created, setCreated] = useState<ClientCreateResponse | null>(null);

  const mutation = useMutation({
    mutationFn: () =>
      clientsApi.create(tenantId, {
        name,
        grant_types: ["authorization_code", "refresh_token"],
        response_types: ["code"],
        token_endpoint_auth_method: authMethod,
        require_pkce: requirePkce,
        redirect_uris: redirectURIs.filter((u) => u.trim() !== ""),
        post_logout_redirect_uris: postLogoutRedirectURIs.filter((u) => u.trim() !== ""),
      }),
    onSuccess: (data) => setCreated(data),
  });

  if (created) {
    return (
      <div className="max-w-lg">
        <PageHeader title="Client Created" />
        <Alert variant="warning">
          <p className="font-medium">
            Client Secret is shown only once. Copy it now.
          </p>
        </Alert>
        <div className="bg-white rounded-lg border border-gray-200 p-6 space-y-3">
          <div>
            <label className="block text-sm text-gray-500 mb-1">
              Client ID
            </label>
            <code className="block bg-gray-50 px-3 py-2 rounded text-sm font-mono break-all">
              {created.client_id}
            </code>
          </div>
          <div>
            <label className="block text-sm text-gray-500 mb-1">
              Client Secret
            </label>
            <code className="block bg-gray-50 px-3 py-2 rounded text-sm font-mono break-all">
              {created.client_secret}
            </code>
          </div>
        </div>
        <div className="mt-4 flex gap-3">
          <Link
            href={`${routes.management.clientDetail(created.id)}`}
            className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
          >
            View Client
          </Link>
          <Link
            href={`${routes.management.tenantClients(tenantId)}`}
            className="px-4 py-2 border border-gray-300 text-sm rounded text-gray-700 hover:bg-gray-50"
          >
            Back to List
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-lg">
      <Link
        href={`${routes.management.tenantClients(tenantId)}`}
        className="text-sm text-blue-600 hover:underline"
      >
        &larr; Back to Clients
      </Link>
      <PageHeader title="Create Client" />

      {mutation.error && (
        <Alert variant="error">{getErrorMessage(mutation.error)}</Alert>
      )}

      <form
        onSubmit={(e) => {
          e.preventDefault();
          mutation.mutate();
        }}
        className="bg-white rounded-lg border border-gray-200 p-6 space-y-4"
      >
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Name
          </label>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            required
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Token Endpoint Auth Method
          </label>
          <select
            value={authMethod}
            onChange={(e) => setAuthMethod(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="client_secret_basic">client_secret_basic</option>
            <option value="client_secret_post">client_secret_post</option>
            <option value="none">none (Public Client)</option>
          </select>
        </div>

        <div className="flex items-center gap-2">
          <input
            id="require_pkce"
            type="checkbox"
            checked={requirePkce}
            onChange={(e) => setRequirePkce(e.target.checked)}
            className="rounded border-gray-300"
          />
          <label htmlFor="require_pkce" className="text-sm text-gray-700">
            Require PKCE
          </label>
        </div>

        <URIListField
          label="Redirect URIs"
          values={redirectURIs}
          onChange={setRedirectURIs}
        />

        <URIListField
          label="Post-Logout Redirect URIs"
          values={postLogoutRedirectURIs}
          onChange={setPostLogoutRedirectURIs}
          placeholder="https://example.com/logout-callback"
        />

        <div className="flex gap-3 pt-2">
          <button
            type="submit"
            disabled={mutation.isPending}
            className="px-4 py-2 bg-blue-600 text-white text-sm rounded hover:bg-blue-700 disabled:opacity-50"
          >
            {mutation.isPending ? "Creating..." : "Create"}
          </button>
          <button
            type="button"
            onClick={() => history.back()}
            className="px-4 py-2 border border-gray-300 text-sm rounded text-gray-700 hover:bg-gray-50"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  );
}
