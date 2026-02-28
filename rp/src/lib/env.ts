function requireEnv(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`環境変数 ${name} が設定されていません`);
  }
  return value;
}

export function getEnv() {
  return {
    databaseUrl: requireEnv("RP_DATABASE_URL"),
    oidc: {
      issuer: requireEnv("RP_OIDC_ISSUER"),
      // コンテナ間通信用 URL（未設定時は issuer にフォールバック）
      issuerInternal: process.env.RP_OIDC_ISSUER_INTERNAL || requireEnv("RP_OIDC_ISSUER"),
      clientId: requireEnv("RP_OIDC_CLIENT_ID"),
      clientSecret: requireEnv("RP_OIDC_CLIENT_SECRET"),
      redirectUri: requireEnv("RP_OIDC_REDIRECT_URI"),
      tenantCode: requireEnv("RP_OIDC_TENANT_CODE"),
      postLogoutRedirectUri: requireEnv("RP_OIDC_POST_LOGOUT_REDIRECT_URI"),
    },
    sessionSecret: requireEnv("RP_SESSION_SECRET"),
    tokenEncryptionKey: requireEnv("RP_TOKEN_ENCRYPTION_KEY"),
  };
}
