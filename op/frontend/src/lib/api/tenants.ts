import type { ListResponse, Tenant } from "@/types";
import { managementFetch } from "@/lib/fetcher";

export const tenantsApi = {
  list(limit = 50, offset = 0) {
    return managementFetch<ListResponse<Tenant>>(
      `/management/v1/tenants?limit=${limit}&offset=${offset}`,
    );
  },

  get(tenantId: string) {
    return managementFetch<Tenant>(`/management/v1/tenants/${tenantId}`);
  },

  create(body: {
    code: string;
    name: string;
    session_lifetime?: number;
    auth_code_lifetime?: number;
    access_token_lifetime?: number;
    refresh_token_lifetime?: number;
    id_token_lifetime?: number;
  }) {
    return managementFetch<Tenant>("/management/v1/tenants", {
      method: "POST",
      body: JSON.stringify(body),
    });
  },

  update(
    tenantId: string,
    body: {
      name?: string;
      session_lifetime?: number;
      auth_code_lifetime?: number;
      access_token_lifetime?: number;
      refresh_token_lifetime?: number;
      id_token_lifetime?: number;
    },
  ) {
    return managementFetch<Tenant>(`/management/v1/tenants/${tenantId}`, {
      method: "PUT",
      body: JSON.stringify(body),
    });
  },
};
