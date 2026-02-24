import type { AdminUser } from "@/types";
import { managementFetch } from "@/lib/fetcher";

export const authApi = {
  login(loginId: string, password: string) {
    return managementFetch<{ user: AdminUser }>(
      "/management/v1/auth/login",
      {
        method: "POST",
        body: JSON.stringify({ login_id: loginId, password }),
      },
    );
  },

  me() {
    return managementFetch<{ user: AdminUser }>("/management/v1/auth/me");
  },

  logout() {
    return managementFetch<void>("/management/v1/auth/logout", {
      method: "POST",
    });
  },
};
