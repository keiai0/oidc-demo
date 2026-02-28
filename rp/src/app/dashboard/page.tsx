import { redirect } from "next/navigation";
import { getSessionId } from "@/lib/session";
import { getSessionWithUser } from "@/lib/db/queries/session";
import { fetchUserInfo } from "@/lib/oidc/userinfo";
import { TokenViewer } from "@/components/token-viewer";
import { UserInfoViewer } from "@/components/userinfo-viewer";
import { SessionInfo } from "@/components/session-info";
import { LogoutButton } from "@/components/logout-button";

export const dynamic = "force-dynamic";

export default async function DashboardPage() {
  const sessionId = await getSessionId();
  if (!sessionId) redirect("/");

  const result = await getSessionWithUser(sessionId);
  if (!result) redirect("/");

  const { session, user } = result;

  // UserInfo エンドポイントからリアルタイム取得
  let userInfo: Record<string, unknown> | null = null;
  let userInfoError: string | null = null;
  try {
    userInfo = await fetchUserInfo(session.accessToken);
  } catch (e) {
    userInfoError = e instanceof Error ? e.message : "取得に失敗しました";
  }

  return (
    <div className="max-w-4xl mx-auto px-4 py-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">ダッシュボード</h1>
          <p className="text-gray-500 text-sm mt-1">
            認証済み — {user.name ?? user.email ?? user.opSub}
          </p>
        </div>
        <LogoutButton />
      </div>

      <div className="space-y-6">
        <SessionInfo
          sessionId={session.id}
          opSessionId={session.opSessionId}
          userId={user.id}
          opSub={user.opSub}
          email={user.email}
          name={user.name}
          tokenExpiresAt={session.tokenExpiresAt.toISOString()}
          sessionExpiresAt={session.expiresAt.toISOString()}
        />

        <TokenViewer title="ID トークン" token={session.idToken} />

        <TokenViewer title="アクセストークン" token={session.accessToken} />

        <UserInfoViewer data={userInfo} error={userInfoError} />
      </div>
    </div>
  );
}
