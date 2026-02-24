export const queryKeys = {
  tenants: {
    all: ["tenants"] as const,
    list: (limit?: number, offset?: number) =>
      ["tenants", "list", { limit, offset }] as const,
    detail: (id: string) => ["tenants", "detail", id] as const,
  },
  clients: {
    all: ["clients"] as const,
    listByTenant: (tenantId: string) =>
      ["clients", "list", { tenantId }] as const,
    detail: (id: string) => ["clients", "detail", id] as const,
  },
  keys: {
    all: ["keys"] as const,
    list: () => ["keys", "list"] as const,
  },
};
