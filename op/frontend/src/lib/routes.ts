export const routes = {
  login: "/login",
  logout: "/logout",
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
    initialAccessTokens: "/management/initial-access-tokens",
    keys: "/management/keys",
    incidents: "/management/incidents",
    federationProviders: "/management/federation-providers",
    federationProviderNew: (tenantId: string) =>
      `/management/federation-providers/new?tenant_id=${tenantId}`,
    federationProviderDetail: (id: string) =>
      `/management/federation-providers/detail?id=${id}`,
  },
} as const;
