import type {
  Client,
  ClientCreateResponse,
  ClientDetail,
  ListResponse,
  RedirectURI,
  RotateSecretResponse,
} from "@/types";
import { managementFetch } from "@/lib/fetcher";

export const clientsApi = {
  listByTenant(tenantId: string, limit = 50, offset = 0) {
    return managementFetch<ListResponse<Client>>(
      `/management/v1/tenants/${tenantId}/clients?limit=${limit}&offset=${offset}`,
    );
  },

  get(clientDbId: string) {
    return managementFetch<ClientDetail>(
      `/management/v1/clients/${clientDbId}`,
    );
  },

  create(
    tenantId: string,
    body: {
      name: string;
      grant_types: string[];
      response_types: string[];
      token_endpoint_auth_method?: string;
      require_pkce?: boolean;
      redirect_uris?: string[];
      post_logout_redirect_uris?: string[];
      frontchannel_logout_uri?: string;
      backchannel_logout_uri?: string;
    },
  ) {
    return managementFetch<ClientCreateResponse>(
      `/management/v1/tenants/${tenantId}/clients`,
      { method: "POST", body: JSON.stringify(body) },
    );
  },

  update(
    clientDbId: string,
    body: {
      name?: string;
      grant_types?: string[];
      response_types?: string[];
      token_endpoint_auth_method?: string;
      require_pkce?: boolean;
      frontchannel_logout_uri?: string;
      backchannel_logout_uri?: string;
    },
  ) {
    return managementFetch<Client>(`/management/v1/clients/${clientDbId}`, {
      method: "PUT",
      body: JSON.stringify(body),
    });
  },

  delete(clientDbId: string) {
    return managementFetch<void>(`/management/v1/clients/${clientDbId}`, {
      method: "DELETE",
    });
  },

  rotateSecret(clientDbId: string) {
    return managementFetch<RotateSecretResponse>(
      `/management/v1/clients/${clientDbId}/secret`,
      { method: "PUT" },
    );
  },

  addRedirectURI(clientDbId: string, uri: string) {
    return managementFetch<RedirectURI>(
      `/management/v1/clients/${clientDbId}/redirect-uris`,
      { method: "POST", body: JSON.stringify({ uri }) },
    );
  },

  deleteRedirectURI(clientDbId: string, uriId: string) {
    return managementFetch<void>(
      `/management/v1/clients/${clientDbId}/redirect-uris/${uriId}`,
      { method: "DELETE" },
    );
  },

  addPostLogoutRedirectURI(clientDbId: string, uri: string) {
    return managementFetch<RedirectURI>(
      `/management/v1/clients/${clientDbId}/post-logout-redirect-uris`,
      { method: "POST", body: JSON.stringify({ uri }) },
    );
  },

  deletePostLogoutRedirectURI(clientDbId: string, uriId: string) {
    return managementFetch<void>(
      `/management/v1/clients/${clientDbId}/post-logout-redirect-uris/${uriId}`,
      { method: "DELETE" },
    );
  },
};
