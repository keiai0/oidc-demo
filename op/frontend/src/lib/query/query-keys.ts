export const queryKeys = {
  tenants: {
    all: ["tenants"] as const,
    list: (limit?: number, offset?: number) =>
      ["tenants", "list", { limit, offset }] as const,
    detail: (id: string) => ["tenants", "detail", id] as const,
  },
  clients: {
    all: ["clients"] as const,
    list: (limit?: number, offset?: number) =>
      ["clients", "list", { limit, offset }] as const,
    listByTenant: (tenantId: string) =>
      ["clients", "list", { tenantId }] as const,
    detail: (id: string) => ["clients", "detail", id] as const,
    tenants: (id: string) => ["clients", "tenants", id] as const,
    tokenExchangePolicy: (id: string) =>
      ["clients", "token-exchange-policy", id] as const,
  },
  keys: {
    all: ["keys"] as const,
    list: () => ["keys", "list"] as const,
  },
  federationProviders: {
    all: ["federation-providers"] as const,
    listByTenant: (tenantId: string) =>
      ["federation-providers", "list", { tenantId }] as const,
    detail: (id: string) => ["federation-providers", "detail", id] as const,
  },
};
