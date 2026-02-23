import { z } from "zod";

const lifetimeField = z.number().int().positive().optional();

export const createTenantSchema = z.object({
  code: z
    .string()
    .min(3, "Code must be at least 3 characters")
    .max(63, "Code must be at most 63 characters")
    .regex(
      /^[a-z0-9][a-z0-9-]*[a-z0-9]$/,
      "Lowercase alphanumeric and hyphens, must start/end with alphanumeric",
    ),
  name: z
    .string()
    .min(1, "Name is required")
    .max(255, "Name must be at most 255 characters"),
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
    .min(1, "Name is required")
    .max(255, "Name must be at most 255 characters"),
  session_lifetime: z.number().int().positive(),
  auth_code_lifetime: z.number().int().positive(),
  access_token_lifetime: z.number().int().positive(),
  refresh_token_lifetime: z.number().int().positive(),
  id_token_lifetime: z.number().int().positive(),
});

export type UpdateTenantInput = z.infer<typeof updateTenantSchema>;
