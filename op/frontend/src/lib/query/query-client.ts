import { QueryClient } from "@tanstack/react-query";
import { ApiRequestError } from "@/lib/fetcher";

export function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30 * 1000,
        retry(failureCount, error) {
          // クライアントエラーではリトライしない
          if (error instanceof ApiRequestError) {
            if ([401, 403, 404].includes(error.status)) return false;
          }
          return failureCount < 2;
        },
      },
    },
  });
}
