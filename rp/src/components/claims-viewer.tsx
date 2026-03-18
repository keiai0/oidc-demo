interface ClaimsViewerProps {
  claimsRequest: Record<string, unknown> | null;
  idTokenClaims: Record<string, unknown>;
  userInfo: Record<string, unknown> | null;
}

/** claims パラメータで要求したクレームと、実際に取得したクレームを比較表示する */
export function ClaimsViewer({
  claimsRequest,
  idTokenClaims,
  userInfo,
}: ClaimsViewerProps) {
  if (!claimsRequest) return null;

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <h2 className="text-lg font-bold mb-4">
        Claims パラメータ（要求 vs 取得）
      </h2>

      <div className="space-y-4">
        <div>
          <h3 className="text-sm font-semibold text-gray-500 mb-1">
            要求した claims パラメータ
          </h3>
          <pre className="bg-gray-50 rounded p-3 text-xs font-mono overflow-x-auto">
            {JSON.stringify(claimsRequest, null, 2)}
          </pre>
        </div>

        {renderTarget("id_token", claimsRequest, idTokenClaims)}
        {renderTarget("userinfo", claimsRequest, userInfo)}
      </div>
    </div>
  );
}

function renderTarget(
  target: string,
  claimsRequest: Record<string, unknown>,
  actualClaims: Record<string, unknown> | null,
) {
  const requested = claimsRequest[target] as Record<string, unknown> | undefined;
  if (!requested) return null;

  const claimNames = Object.keys(requested);

  return (
    <div>
      <h3 className="text-sm font-semibold text-gray-500 mb-2">{target}</h3>
      <table className="w-full text-xs">
        <thead>
          <tr className="text-left text-gray-500 border-b">
            <th className="py-1 pr-3">クレーム</th>
            <th className="py-1 pr-3">essential</th>
            <th className="py-1 pr-3">取得値</th>
            <th className="py-1">状態</th>
          </tr>
        </thead>
        <tbody>
          {claimNames.map((name) => {
            const spec = requested[name] as { essential?: boolean } | null;
            const isEssential = spec?.essential === true;
            const value = actualClaims?.[name];
            const received = value !== undefined;

            return (
              <tr key={name} className="border-b border-gray-100">
                <td className="py-1.5 pr-3">
                  <code className="bg-gray-100 px-1 rounded">{name}</code>
                </td>
                <td className="py-1.5 pr-3">
                  {isEssential ? (
                    <span className="text-orange-600 font-medium">yes</span>
                  ) : (
                    <span className="text-gray-400">no</span>
                  )}
                </td>
                <td className="py-1.5 pr-3 font-mono max-w-[200px] truncate">
                  {received ? String(value) : "-"}
                </td>
                <td className="py-1.5">
                  {received ? (
                    <span className="text-green-600">取得済み</span>
                  ) : (
                    <span className="text-red-500">未取得</span>
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
