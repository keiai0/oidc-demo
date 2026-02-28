import { z } from "zod";

export const revokeByTenantSchema = z.object({
  tenant_id: z.string().uuid("テナントIDの形式が無効です"),
});

export type RevokeByTenantInput = z.infer<typeof revokeByTenantSchema>;

export const revokeByUserSchema = z.object({
  user_id: z.string().uuid("ユーザーIDの形式が無効です"),
});

export type RevokeByUserInput = z.infer<typeof revokeByUserSchema>;
