export type Tenant = {
  id: string;
  code: string;
  name: string;
  session_lifetime: number;
  auth_code_lifetime: number;
  access_token_lifetime: number;
  refresh_token_lifetime: number;
  id_token_lifetime: number;
  created_at: string;
  updated_at: string;
};
