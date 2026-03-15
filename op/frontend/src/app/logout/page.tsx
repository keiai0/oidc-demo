"use client";

import { useState, useEffect, useCallback, useRef } from "react";

/**
 * ログアウト完了ページ
 *
 * OP バックエンドの GET /{tenant_code}/logout からリダイレクトされる。
 * Front-Channel Logout 対象の RP がある場合、iframe で通知してからリダイレクトする。
 *
 * クエリパラメータ:
 * - frontchannel_uris: カンマ区切りの RP frontchannel_logout_uri 一覧
 * - iss: OP の issuer 識別子
 * - sid: セッション ID
 * - post_logout_redirect_uri: ログアウト完了後のリダイレクト先
 * - state: CSRF 対策用の state パラメータ
 *
 * 仕様参照: OIDC Front-Channel Logout 1.0
 */
export default function LogoutPage() {
  const [status, setStatus] = useState<"loading" | "done">("loading");
  const [frontchannelURIs, setFrontchannelURIs] = useState<string[]>([]);
  const [postLogoutRedirectURI, setPostLogoutRedirectURI] = useState("");
  const [stateParam, setStateParam] = useState("");
  const [issuer, setIssuer] = useState("");
  const [sid, setSid] = useState("");
  const loadedCount = useRef(0);
  const redirected = useRef(false);

  const doRedirect = useCallback(() => {
    if (redirected.current) return;
    redirected.current = true;

    if (postLogoutRedirectURI) {
      const url = new URL(postLogoutRedirectURI);
      if (stateParam) {
        url.searchParams.set("state", stateParam);
      }
      window.location.href = url.toString();
    } else {
      setStatus("done");
    }
  }, [postLogoutRedirectURI, stateParam]);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const urisParam = params.get("frontchannel_uris") || "";
    const uris = urisParam ? urisParam.split(",").filter(Boolean) : [];
    const postLogout = params.get("post_logout_redirect_uri") || "";
    const state = params.get("state") || "";
    const iss = params.get("iss") || "";
    const sidParam = params.get("sid") || "";

    setFrontchannelURIs(uris);
    setPostLogoutRedirectURI(postLogout);
    setStateParam(state);
    setIssuer(iss);
    setSid(sidParam);

    if (uris.length === 0) {
      // Front-Channel 対象なし → 即リダイレクトまたは完了表示
      if (postLogout) {
        const url = new URL(postLogout);
        if (state) {
          url.searchParams.set("state", state);
        }
        window.location.href = url.toString();
      } else {
        setStatus("done");
      }
    }
  }, []);

  // Front-Channel iframe のロード完了 or タイムアウト後にリダイレクト
  useEffect(() => {
    if (frontchannelURIs.length === 0) return;

    // 3秒タイムアウト: iframe がロードされなくてもリダイレクトする
    const timer = setTimeout(() => {
      doRedirect();
    }, 3000);

    return () => clearTimeout(timer);
  }, [frontchannelURIs, doRedirect]);

  const handleIframeLoad = () => {
    loadedCount.current += 1;
    if (loadedCount.current >= frontchannelURIs.length) {
      doRedirect();
    }
  };

  // iframe の src を構築: iss と sid は両方セットで送るか送らないか
  // (OIDC Front-Channel Logout 1.0: "If either is included, both MUST be.")
  const buildIframeSrc = (uri: string): string => {
    const url = new URL(uri);
    if (issuer && sid) {
      url.searchParams.set("iss", issuer);
      url.searchParams.set("sid", sid);
    }
    return url.toString();
  };

  return (
    <div
      style={{
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
        minHeight: "100vh",
        fontFamily: "sans-serif",
      }}
    >
      {status === "loading" ? (
        <p>ログアウト中...</p>
      ) : (
        <div style={{ textAlign: "center" }}>
          <p>ログアウトしました。</p>
        </div>
      )}

      {/* Front-Channel Logout 用の hidden iframe */}
      {frontchannelURIs.map((uri) => (
        <iframe
          key={uri}
          src={buildIframeSrc(uri)}
          style={{ display: "none", width: 0, height: 0, border: "none" }}
          onLoad={handleIframeLoad}
          onError={handleIframeLoad}
          title={`logout-${uri}`}
          sandbox=""
        />
      ))}
    </div>
  );
}
