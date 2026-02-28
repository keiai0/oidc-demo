interface UserInfoViewerProps {
  data: Record<string, unknown> | null;
  error?: string | null;
}

export function UserInfoViewer({ data, error }: UserInfoViewerProps) {
  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <h2 className="text-lg font-bold mb-4">UserInfo エンドポイント</h2>

      {error ? (
        <p className="text-red-600 text-sm">取得エラー: {error}</p>
      ) : data ? (
        <pre className="bg-gray-50 rounded p-3 text-xs font-mono overflow-x-auto">
          {JSON.stringify(data, null, 2)}
        </pre>
      ) : (
        <p className="text-gray-400 text-sm">データなし</p>
      )}
    </div>
  );
}
