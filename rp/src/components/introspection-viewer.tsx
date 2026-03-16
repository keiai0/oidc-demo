interface IntrospectionViewerProps {
  data: Record<string, unknown> | null;
  error: string | null;
}

export function IntrospectionViewer({ data, error }: IntrospectionViewerProps) {
  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <h2 className="text-lg font-bold mb-4">Token Introspection (RFC 7662)</h2>
      {error ? (
        <p className="text-red-600 text-sm">{error}</p>
      ) : data ? (
        <div>
          <div className="mb-2">
            <span
              className={`inline-block px-2 py-1 text-xs font-semibold rounded ${
                data.active
                  ? "bg-green-100 text-green-800"
                  : "bg-red-100 text-red-800"
              }`}
            >
              {data.active ? "active" : "inactive"}
            </span>
          </div>
          <pre className="bg-gray-50 p-4 rounded text-xs overflow-x-auto">
            {JSON.stringify(data, null, 2)}
          </pre>
        </div>
      ) : (
        <p className="text-gray-400 text-sm">データなし</p>
      )}
    </div>
  );
}
