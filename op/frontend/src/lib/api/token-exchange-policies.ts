import type { TokenExchangePolicy } from "@/types";
import { managementFetch } from "@/lib/fetcher";

export const tokenExchangePoliciesApi = {
  get(clientDbId: string) {
    return managementFetch<TokenExchangePolicy>(
      `/management/v1/clients/${clientDbId}/token-exchange-policy`,
    );
  },

  createOrUpdate(
    clientDbId: string,
    body: {
      allowed_subject_token_types: string[];
      allowed_requested_token_types: string[];
      allowed_audiences: string[];
      allow_impersonation: boolean;
      allow_delegation: boolean;
    },
  ) {
    return managementFetch<TokenExchangePolicy>(
      `/management/v1/clients/${clientDbId}/token-exchange-policy`,
      { method: "PUT", body: JSON.stringify(body) },
    );
  },

  delete(clientDbId: string) {
    return managementFetch<void>(
      `/management/v1/clients/${clientDbId}/token-exchange-policy`,
      { method: "DELETE" },
    );
  },
};
