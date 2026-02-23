import { z } from "zod";

export const createClientSchema = z.object({
  name: z
    .string()
    .min(1, "Name is required")
    .max(255, "Name must be at most 255 characters"),
  grant_types: z.array(z.string()).min(1, "At least one grant type is required"),
  response_types: z
    .array(z.string())
    .min(1, "At least one response type is required"),
  token_endpoint_auth_method: z.string().default("client_secret_basic"),
  require_pkce: z.boolean().default(true),
  redirect_uris: z.array(z.string().url("Invalid URI")).default([]),
  post_logout_redirect_uris: z.array(z.string().url("Invalid URI")).default([]),
  frontchannel_logout_uri: z.string().url("Invalid URI").optional().or(z.literal("")),
  backchannel_logout_uri: z.string().url("Invalid URI").optional().or(z.literal("")),
});

export type CreateClientInput = z.infer<typeof createClientSchema>;
