import { z } from "zod";

export const createClientSchema = z.object({
  name: z
    .string()
    .min(1, "名前を入力してください")
    .max(255, "名前は255文字以内で入力してください"),
  grant_types: z.array(z.string()).min(1, "Grant Type を1つ以上選択してください"),
  response_types: z
    .array(z.string())
    .min(1, "Response Type を1つ以上選択してください"),
  token_endpoint_auth_method: z.string().default("client_secret_basic"),
  require_pkce: z.boolean().default(true),
  redirect_uris: z.array(z.string().url("無効な URI です")).default([]),
  post_logout_redirect_uris: z.array(z.string().url("無効な URI です")).default([]),
  frontchannel_logout_uri: z.string().url("無効な URI です").optional().or(z.literal("")),
  backchannel_logout_uri: z.string().url("無効な URI です").optional().or(z.literal("")),
});

export type CreateClientInput = z.infer<typeof createClientSchema>;
