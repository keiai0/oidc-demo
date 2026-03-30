import { getEnv } from "../env";

export interface TokenExchangeRequest {
  subjectToken: string;
  mode: "impersonation" | "delegation";
  actorToken?: string;
  audience?: string;
  scope?: string;
}

export interface TokenExchangeResult {
  access_token: string;
  issued_token_type: string;
  token_type: string;
  expires_in: number;
  scope?: string;
}

/**
 * Token Exchange (RFC 8693) リクエストを OP に送信する。
 * Token Exchange 用のサービスクライアント（demo-service）の認証情報を使用する。
 */
export async function performTokenExchange(
  req: TokenExchangeRequest,
): Promise<TokenExchangeResult> {
  const env = getEnv();
  const issuerInternal = env.oidc.issuerInternal;
  const tenantCode = env.oidc.tenantCode;
  const tokenEndpoint = `${issuerInternal}/${tenantCode}/token`;

  // Token Exchange 用のサービスクライアント認証情報
  const serviceClientId =
    process.env.RP_TOKEN_EXCHANGE_CLIENT_ID ?? "demo-service";
  const serviceClientSecret =
    process.env.RP_TOKEN_EXCHANGE_CLIENT_SECRET ?? "demo-rp-secret";

  const body = new URLSearchParams({
    grant_type: "urn:ietf:params:oauth:grant-type:token-exchange",
    subject_token: req.subjectToken,
    subject_token_type: "urn:ietf:params:oauth:token-type:access_token",
    requested_token_type: "urn:ietf:params:oauth:token-type:access_token",
  });

  if (req.audience) {
    body.set("audience", req.audience);
  }
  if (req.scope) {
    body.set("scope", req.scope);
  }

  // Delegation の場合は actor_token を追加
  if (req.mode === "delegation" && req.actorToken) {
    body.set("actor_token", req.actorToken);
    body.set("actor_token_type", "urn:ietf:params:oauth:token-type:access_token");
  }

  const res = await fetch(tokenEndpoint, {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
      Authorization:
        "Basic " +
        Buffer.from(`${serviceClientId}:${serviceClientSecret}`).toString(
          "base64",
        ),
    },
    body: body.toString(),
  });

  if (!res.ok) {
    const errorBody = await res.json().catch(() => ({ error: "unknown" }));
    throw new Error(
      `Token Exchange failed: ${errorBody.error ?? res.status} ${errorBody.error_description ?? ""}`,
    );
  }

  return res.json();
}
