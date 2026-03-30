export type TokenExchangePolicy = {
  id: string;
  client_id: string;
  allowed_subject_token_types: string[];
  allowed_requested_token_types: string[];
  allowed_audiences: string[];
  allow_impersonation: boolean;
  allow_delegation: boolean;
  created_at: string;
  updated_at: string;
};
