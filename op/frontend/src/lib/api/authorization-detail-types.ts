import type { AuthorizationDetailType } from "@/types";
import { managementFetch } from "@/lib/fetcher";

export const authorizationDetailTypesApi = {
  list(tenantId: string) {
    return managementFetch<{ data: AuthorizationDetailType[] }>(
      `/management/v1/tenants/${tenantId}/authorization-detail-types`,
    );
  },

  get(id: string) {
    return managementFetch<AuthorizationDetailType>(
      `/management/v1/authorization-detail-types/${id}`,
    );
  },

  create(
    tenantId: string,
    body: {
      type_name: string;
      description: string;
      json_schema?: string;
      allowed_actions: string[];
      allowed_locations: string[];
    },
  ) {
    return managementFetch<AuthorizationDetailType>(
      `/management/v1/tenants/${tenantId}/authorization-detail-types`,
      {
        method: "POST",
        body: JSON.stringify(body),
      },
    );
  },

  update(
    id: string,
    body: {
      type_name?: string;
      description?: string;
      json_schema?: string | null;
      allowed_actions?: string[];
      allowed_locations?: string[];
    },
  ) {
    return managementFetch<AuthorizationDetailType>(
      `/management/v1/authorization-detail-types/${id}`,
      {
        method: "PUT",
        body: JSON.stringify(body),
      },
    );
  },

  delete(id: string) {
    return managementFetch<void>(
      `/management/v1/authorization-detail-types/${id}`,
      { method: "DELETE" },
    );
  },
};
