import { z } from "zod";

export const revokeByTenantSchema = z.object({
  tenant_id: z.string().uuid("Invalid tenant ID format"),
});

export type RevokeByTenantInput = z.infer<typeof revokeByTenantSchema>;

export const revokeByUserSchema = z.object({
  user_id: z.string().uuid("Invalid user ID format"),
});

export type RevokeByUserInput = z.infer<typeof revokeByUserSchema>;
