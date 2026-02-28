export const routes = {
  login: "/login",
  management: {
    root: "/management",
    login: "/management/login",
    tenants: "/management/tenants",
    tenantNew: "/management/tenants/new",
    clients: "/management/clients",
    tenantDetail: (id: string) => `/management/tenants/detail?id=${id}`,
    tenantClients: (tenantId: string) =>
      `/management/tenants/detail/clients?tenant_id=${tenantId}`,
    tenantClientNew: (tenantId: string) =>
      `/management/tenants/detail/clients/new?tenant_id=${tenantId}`,
    clientDetail: (id: string) => `/management/clients/detail?id=${id}`,
    keys: "/management/keys",
    incidents: "/management/incidents",
  },
} as const;
