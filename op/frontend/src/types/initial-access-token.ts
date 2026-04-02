export type InitialAccessToken = {
  id: string;
  tenant_id: string;
  max_registrations: number;
  used_count: number;
  expires_at: string;
  revoked_at?: string;
  created_at: string;
};

export type InitialAccessTokenCreateResponse = InitialAccessToken & {
  token: string;
};
