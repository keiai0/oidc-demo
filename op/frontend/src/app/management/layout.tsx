"use client";

import { useState } from "react";
import { QueryClientProvider } from "@tanstack/react-query";
import { ManagementAuthProvider } from "@/lib/management-auth";
import { createQueryClient } from "@/lib/query/query-client";

export default function ManagementLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const [queryClient] = useState(() => createQueryClient());

  return (
    <QueryClientProvider client={queryClient}>
      <ManagementAuthProvider>{children}</ManagementAuthProvider>
    </QueryClientProvider>
  );
}
