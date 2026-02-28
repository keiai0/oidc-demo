interface SessionInfoProps {
  sessionId: string;
  opSessionId: string | null;
  userId: string;
  opSub: string;
  email: string | null;
  name: string | null;
  tokenExpiresAt: string;
  sessionExpiresAt: string;
}

export function SessionInfo(props: SessionInfoProps) {
  const rows = [
    { label: "RP セッション ID", value: props.sessionId },
    { label: "OP セッション ID (sid)", value: props.opSessionId ?? "-" },
    { label: "RP ユーザー ID", value: props.userId },
    { label: "OP Subject (sub)", value: props.opSub },
    { label: "メール", value: props.email ?? "-" },
    { label: "名前", value: props.name ?? "-" },
    {
      label: "トークン有効期限",
      value: new Date(props.tokenExpiresAt).toLocaleString("ja-JP", { timeZone: "Asia/Tokyo" }),
    },
    {
      label: "セッション有効期限",
      value: new Date(props.sessionExpiresAt).toLocaleString("ja-JP", { timeZone: "Asia/Tokyo" }),
    },
  ];

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <h2 className="text-lg font-bold mb-4">セッション情報</h2>
      <dl className="space-y-2">
        {rows.map((row) => (
          <div key={row.label} className="flex flex-col sm:flex-row sm:gap-4">
            <dt className="text-sm font-semibold text-gray-500 sm:w-48 shrink-0">
              {row.label}
            </dt>
            <dd className="text-sm font-mono break-all">{row.value}</dd>
          </div>
        ))}
      </dl>
    </div>
  );
}
