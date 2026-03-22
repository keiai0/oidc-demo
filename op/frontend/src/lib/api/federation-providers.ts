import type { FederationProvider } from "@/types";
import { managementFetch } from "@/lib/fetcher";

export const federationProvidersApi = {
  list(tenantId: string) {
    return managementFetch<{ data: FederationProvider[] }>(
      `/management/v1/tenants/${tenantId}/federation-providers`,
    );
  },

  get(id: string) {
    return managementFetch<FederationProvider>(
      `/management/v1/federation-providers/${id}`,
    );
  },

  create(
    tenantId: string,
    body: {
      name: string;
      issuer: string;
      client_id: string;
      client_secret: string;
      scopes?: string;
      auto_provision?: boolean;
    },
  ) {
    return managementFetch<FederationProvider>(
      `/management/v1/tenants/${tenantId}/federation-providers`,
      {
        method: "POST",
        body: JSON.stringify(body),
      },
    );
  },

  update(
    id: string,
    body: {
      issuer?: string;
      client_id?: string;
      client_secret?: string;
      scopes?: string;
      auto_provision?: boolean;
      status?: string;
    },
  ) {
    return managementFetch<FederationProvider>(
      `/management/v1/federation-providers/${id}`,
      {
        method: "PUT",
        body: JSON.stringify(body),
      },
    );
  },

  delete(id: string) {
    return managementFetch<void>(`/management/v1/federation-providers/${id}`, {
      method: "DELETE",
    });
  },
};
