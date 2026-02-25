export const routes = {
  login: "/login",
  management: {
    root: "/management",
    login: "/management/login",
    tenants: "/management/tenants",
    tenantNew: "/management/tenants/new",
    tenantDetail: (id: string) => `/management/tenants/${id}`,
    tenantClients: (tenantId: string) =>
      `/management/tenants/${tenantId}/clients`,
    tenantClientNew: (tenantId: string) =>
      `/management/tenants/${tenantId}/clients/new`,
    clientDetail: (id: string) => `/management/clients/${id}`,
    keys: "/management/keys",
    incidents: "/management/incidents",
  },
} as const;
