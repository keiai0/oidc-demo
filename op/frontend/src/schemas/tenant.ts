import { z } from "zod";

const lifetimeField = z.number().int().positive().optional();

export const createTenantSchema = z.object({
  code: z
    .string()
    .min(3, "コードは3文字以上で入力してください")
    .max(63, "コードは63文字以内で入力してください")
    .regex(
      /^[a-z0-9][a-z0-9-]*[a-z0-9]$/,
      "小文字英数字とハイフンのみ。先頭・末尾は英数字にしてください",
    ),
  name: z
    .string()
    .min(1, "名前を入力してください")
    .max(255, "名前は255文字以内で入力してください"),
  session_lifetime: lifetimeField,
  auth_code_lifetime: lifetimeField,
  access_token_lifetime: lifetimeField,
  refresh_token_lifetime: lifetimeField,
  id_token_lifetime: lifetimeField,
});

export type CreateTenantInput = z.infer<typeof createTenantSchema>;

export const updateTenantSchema = z.object({
  name: z
    .string()
    .min(1, "名前を入力してください")
    .max(255, "名前は255文字以内で入力してください"),
  session_lifetime: z.number().int().positive(),
  auth_code_lifetime: z.number().int().positive(),
  access_token_lifetime: z.number().int().positive(),
  refresh_token_lifetime: z.number().int().positive(),
  id_token_lifetime: z.number().int().positive(),
});

export type UpdateTenantInput = z.infer<typeof updateTenantSchema>;
