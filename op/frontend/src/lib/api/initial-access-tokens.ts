import type {
  InitialAccessToken,
  InitialAccessTokenCreateResponse,
} from "@/types/initial-access-token";
import type { ListResponse } from "@/types";
import { managementFetch } from "@/lib/fetcher";

export const initialAccessTokensApi = {
  list(tenantId: string) {
    return managementFetch<ListResponse<InitialAccessToken>>(
      `/management/v1/tenants/${tenantId}/initial-access-tokens`,
    );
  },

  create(
    tenantId: string,
    body: { max_registrations: number; expires_in: number },
  ) {
    return managementFetch<InitialAccessTokenCreateResponse>(
      `/management/v1/tenants/${tenantId}/initial-access-tokens`,
      { method: "POST", body: JSON.stringify(body) },
    );
  },

  revoke(id: string) {
    return managementFetch<void>(
      `/management/v1/initial-access-tokens/${id}`,
      { method: "DELETE" },
    );
  },
};
