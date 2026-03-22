export type FederationProvider = {
  id: string;
  tenant_id: string;
  name: string;
  issuer: string;
  client_id: string;
  scopes: string;
  auto_provision: boolean;
  status: string;
  created_at: string;
  updated_at: string;
};
