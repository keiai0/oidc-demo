import type { RevokeResponse } from "@/types";
import { managementFetch } from "@/lib/fetcher";

export const incidentsApi = {
  revokeAll() {
    return managementFetch<RevokeResponse>(
      "/management/v1/incidents/revoke-all-tokens",
      { method: "POST" },
    );
  },

  revokeByTenant(tenantId: string) {
    return managementFetch<RevokeResponse>(
      "/management/v1/incidents/revoke-tenant-tokens",
      { method: "POST", body: JSON.stringify({ tenant_id: tenantId }) },
    );
  },

  revokeByUser(userId: string) {
    return managementFetch<RevokeResponse>(
      "/management/v1/incidents/revoke-user-tokens",
      { method: "POST", body: JSON.stringify({ user_id: userId }) },
    );
  },
};
