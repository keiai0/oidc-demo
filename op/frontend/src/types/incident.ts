export type RevokeResponse = {
  revoked: {
    sessions: number;
    access_tokens: number;
    refresh_tokens: number;
  };
};
