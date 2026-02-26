"use client";

import { useManagementAuth } from "@/lib/management-auth";
import { routes } from "@/lib/routes";
import { FullScreenLoading } from "@/components/ui/loading";
import { Sidebar } from "./sidebar";

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useManagementAuth();

  if (isLoading) {
    return <FullScreenLoading />;
  }

  if (!isAuthenticated) {
    if (typeof window !== "undefined") {
      window.location.href = routes.management.login;
    }
    return null;
  }

  return (
    <div className="flex h-screen">
      <Sidebar />
      <main className="flex-1 overflow-y-auto p-6">{children}</main>
    </div>
  );
}
