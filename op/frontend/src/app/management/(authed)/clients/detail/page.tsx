"use client";

import Link from "next/link";
import { Fragment, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { clientsApi } from "@/lib/api/clients";
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

  const { data: client, isLoading } = useQuery({
    queryKey: queryKeys.clients.detail(id),
    queryFn: () => clientsApi.get(id),
    enabled: !!id,
  });

  const invalidateClient = () =>
    queryClient.invalidateQueries({ queryKey: queryKeys.clients.detail(id) });

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
      window.location.href = client
        ? routes.management.tenantClients(client.tenant_id)
        : routes.management.tenants;
    },
    onError: (err) => setError(getErrorMessage(err)),
  });

  const handleRotateSecret = () => {
    if (!confirm("Are you sure? The current secret will be invalidated."))
      return;
    rotateSecretMutation.mutate();
  };

  const handleDelete = () => {
    if (!confirm("Are you sure you want to delete this client?")) return;
    deleteMutation.mutate();
  };

  if (isLoading) return <Loading />;
  if (!client) return <p className="text-gray-500">Client not found.</p>;

  const infoFields = [
    ["Client ID", client.client_id],
    ["Auth Method", client.token_endpoint_auth_method],
    ["Grant Types", client.grant_types.join(", ")],
    ["Response Types", client.response_types.join(", ")],
    ["PKCE", client.require_pkce ? "Required" : "Optional"],
    ["Created", new Date(client.created_at).toLocaleString()],
    ["Updated", new Date(client.updated_at).toLocaleString()],
  ] as const;

  return (
    <div className="max-w-2xl">
      <Link
        href={routes.management.tenantClients(client.tenant_id)}
        className="text-sm text-blue-600 hover:underline"
      >
        &larr; Back to Clients
      </Link>
      <div className="flex items-center justify-between mt-1 mb-6">
        <h1 className="text-2xl font-bold text-gray-900">{client.name}</h1>
        <Badge variant={client.status === "active" ? "active" : "inactive"}>
          {client.status}
        </Badge>
      </div>

      {error && <Alert variant="error">{error}</Alert>}

      {/* Client Info */}
      <Card title="Client Info" className="mb-4">
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

      {/* Secret Rotation */}
      <Card title="Client Secret" className="mb-4">
        {newSecret && (
          <Alert variant="warning">
            <p className="font-medium mb-1">New secret (shown only once):</p>
            <code className="block bg-white px-3 py-2 rounded text-sm font-mono break-all border">
              {newSecret}
            </code>
          </Alert>
        )}
        <button
          onClick={handleRotateSecret}
          className="px-4 py-2 bg-yellow-500 text-white text-sm rounded hover:bg-yellow-600"
        >
          Rotate Secret
        </button>
      </Card>

      {/* Redirect URIs */}
      <Card title="Redirect URIs" className="mb-4">
        {client.redirect_uris.length === 0 ? (
          <p className="text-sm text-gray-500 mb-3">No redirect URIs.</p>
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
                  Delete
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
            Add
          </button>
        </div>
      </Card>

      {/* Post-Logout Redirect URIs */}
      <Card title="Post-Logout Redirect URIs" className="mb-4">
        {client.post_logout_redirect_uris.length === 0 ? (
          <p className="text-sm text-gray-500 mb-3">
            No post-logout redirect URIs.
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
                  Delete
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
            Add
          </button>
        </div>
      </Card>

      {/* Danger Zone */}
      <Card title="Danger Zone" variant="danger">
        <button
          onClick={handleDelete}
          className="px-4 py-2 bg-red-600 text-white text-sm rounded hover:bg-red-700"
        >
          Delete Client
        </button>
      </Card>
    </div>
  );
}
