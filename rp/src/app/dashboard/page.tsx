import { redirect } from "next/navigation";
import { getSessionId } from "@/lib/session";
import { getSessionWithUser } from "@/lib/db/queries/session";
import { introspectToken } from "@/lib/oidc/introspect";
import { TokenViewer } from "@/components/token-viewer";
import { UserInfoViewer } from "@/components/userinfo-viewer";
import { IntrospectionViewer } from "@/components/introspection-viewer";
import { ClaimsViewer } from "@/components/claims-viewer";
import { SessionInfo } from "@/components/session-info";
import { LogoutButton } from "@/components/logout-button";
import { SecurityDemoPanel } from "@/components/security-demo-panel";
import { isDemoMode } from "@/lib/env";
import { decodeJwt } from "jose";

export const dynamic = "force-dynamic";

export default async function DashboardPage() {
  const sessionId = await getSessionId();
  if (!sessionId) redirect("/");

  const result = await getSessionWithUser(sessionId);
  if (!result) redirect("/");

  const { session, user } = result;

  // UserInfo はコールバック時に取得・保存済みのキャッシュを使用
  // DPoP-bound トークンの場合、ダッシュボードから直接 userinfo API を呼べないため
  const userInfo = (session.userinfoJson as Record<string, unknown>) ?? null;

  // Claims パラメータで要求したクレームの比較表示用
  const claimsRequest =
    (session.claimsRequestJson as Record<string, unknown>) ?? null;
  let idTokenClaims: Record<string, unknown> = {};
  try {
    idTokenClaims = decodeJwt(session.idToken) as Record<string, unknown>;
  } catch {
    // デコード失敗時は空オブジェクト
  }

  // Token Introspection: サーバーサイドでトークンの有効性をリアルタイム確認
  let introspectionResult: Record<string, unknown> | null = null;
  let introspectionError: string | null = null;
  try {
    introspectionResult = await introspectToken(session.accessToken);
  } catch (e) {
    introspectionError =
      e instanceof Error ? e.message : "Introspection に失敗しました";
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
        <div className="flex items-center gap-3">
          <a
            href={`${process.env.RP_OP_FRONTEND_URL ?? "http://localhost:3000"}/account`}
            className="text-sm text-blue-600 hover:underline"
          >
            アカウント設定
          </a>
          <LogoutButton />
        </div>
      </div>

      <div className="space-y-6">
        <SessionInfo
          sessionId={session.id}
          opSessionId={session.opSessionId}
          userId={user.id}
          opSub={user.opSub}
          email={user.email}
          name={user.name}
          tokenType={session.tokenType}
          tokenExpiresAt={session.tokenExpiresAt.toISOString()}
          sessionExpiresAt={session.expiresAt.toISOString()}
        />

        <TokenViewer title="ID トークン" token={session.idToken} />

        <TokenViewer title="アクセストークン" token={session.accessToken} />

        <UserInfoViewer data={userInfo} error={null} />

        <ClaimsViewer
          claimsRequest={claimsRequest}
          idTokenClaims={idTokenClaims}
          userInfo={userInfo}
        />

        <IntrospectionViewer data={introspectionResult} error={introspectionError} />

        {isDemoMode() && <SecurityDemoPanel />}
      </div>
    </div>
  );
}
