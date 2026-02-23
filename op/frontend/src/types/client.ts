export type Client = {
  id: string;
  tenant_id: string;
  client_id: string;
  name: string;
  grant_types: string[];
  response_types: string[];
  token_endpoint_auth_method: string;
  require_pkce: boolean;
  frontchannel_logout_uri?: string;
  backchannel_logout_uri?: string;
  status: string;
  created_at: string;
  updated_at: string;
};

export type RedirectURI = {
  id: string;
  uri: string;
  created_at: string;
};

export type ClientDetail = Client & {
  redirect_uris: RedirectURI[];
  post_logout_redirect_uris: RedirectURI[];
};

/** クライアント作成時のみ返される（平文のシークレットを含む）。 */
export type ClientCreateResponse = Client & {
  client_secret: string;
};

export type RotateSecretResponse = {
  client_id: string;
  client_secret: string;
};
