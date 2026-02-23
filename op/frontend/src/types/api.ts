/** 管理 API の標準エラーレスポンス。 */
export type ApiError = {
  error: string;
  error_description: string;
};

/** ページネーション付きリストレスポンスのラッパー。 */
export type ListResponse<T> = {
  data: T[];
  total_count: number;
};
