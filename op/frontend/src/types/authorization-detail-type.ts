export type AuthorizationDetailType = {
  id: string;
  tenant_id: string;
  type_name: string;
  description: string;
  json_schema?: string | null;
  allowed_actions: string[];
  allowed_locations: string[];
  created_at: string;
  updated_at: string;
};
