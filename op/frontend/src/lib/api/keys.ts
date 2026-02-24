import type { SignKey } from "@/types";
import { managementFetch } from "@/lib/fetcher";

export const keysApi = {
  list() {
    return managementFetch<SignKey[]>("/management/v1/keys");
  },

  rotate() {
    return managementFetch<SignKey>("/management/v1/keys/rotate", {
      method: "POST",
    });
  },

  deactivate(kid: string) {
    return managementFetch<void>(`/management/v1/keys/${kid}`, {
      method: "DELETE",
    });
  },
};
