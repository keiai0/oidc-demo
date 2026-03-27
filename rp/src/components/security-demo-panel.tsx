"use client";

import { useEffect, useState, useCallback } from "react";

type DemoConfig = {
  stateEnabled: boolean;
  nonceEnabled: boolean;
  pkceEnabled: boolean;
};

const DEFAULT_CONFIG: DemoConfig = {
  stateEnabled: true,
  nonceEnabled: true,
  pkceEnabled: true,
};

const TOGGLE_ITEMS: {
  key: keyof DemoConfig;
  label: string;
  description: string;
}[] = [
  {
    key: "stateEnabled",
    label: "state",
    description: "CSRF 保護（RFC 6749 §10.12）",
  },
  {
    key: "nonceEnabled",
    label: "nonce",
    description: "ID Token リプレイ保護（OIDC Core §3.1.2.1）",
  },
  {
    key: "pkceEnabled",
    label: "PKCE",
    description: "認可コード横取り保護（RFC 7636）",
  },
];

export function SecurityDemoPanel() {
  const [config, setConfig] = useState<DemoConfig>(DEFAULT_CONFIG);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("/api/demo/config")
      .then((res) => res.json())
      .then((data) => setConfig(data))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const updateConfig = useCallback(
    async (key: keyof DemoConfig, value: boolean) => {
      const updated = { ...config, [key]: value };
      setConfig(updated);

      try {
        await fetch("/api/demo/config", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(updated),
        });
      } catch {
        // ロールバック
        setConfig(config);
      }
    },
    [config],
  );

  if (loading) {
    return (
      <div className="bg-white rounded-lg shadow-md p-6">
        <p className="text-gray-400 text-sm">読み込み中...</p>
      </div>
    );
  }

  const allEnabled = config.stateEnabled && config.nonceEnabled && config.pkceEnabled;

  return (
    <div className="space-y-6">
      <div className="bg-white rounded-lg shadow-md p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold">セキュリティ設定</h2>
          <span className="text-xs px-2 py-1 rounded bg-amber-100 text-amber-700 font-medium">
            デモモード
          </span>
        </div>

        <p className="text-sm text-gray-500 mb-4">
          各セキュリティ機構を個別に無効化して、対応する攻撃が成功する様子を確認できます。
        </p>

        {!allEnabled && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded text-sm text-red-700">
            セキュリティ機構が無効化されています。攻撃に対して脆弱な状態です。
          </div>
        )}

        <div className="space-y-3">
          {TOGGLE_ITEMS.map((item) => (
            <div
              key={item.key}
              className="flex items-center justify-between p-3 bg-gray-50 rounded"
            >
              <div>
                <span className="font-mono text-sm font-medium">{item.label}</span>
                <p className="text-xs text-gray-500 mt-0.5">{item.description}</p>
              </div>
              <button
                type="button"
                role="switch"
                aria-checked={config[item.key]}
                onClick={() => updateConfig(item.key, !config[item.key])}
                className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ${
                  config[item.key] ? "bg-green-500" : "bg-red-400"
                }`}
              >
                <span
                  className={`pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow transform transition-transform duration-200 ${
                    config[item.key] ? "translate-x-5" : "translate-x-0"
                  }`}
                />
              </button>
            </div>
          ))}
        </div>

        {/* 総合比較表 */}
        <details className="mt-4">
          <summary className="text-sm font-medium text-gray-700 cursor-pointer hover:text-gray-900">
            3 つの機構の比較表を見る
          </summary>
          <div className="mt-2 overflow-x-auto">
            <table className="w-full text-xs border border-gray-200 rounded">
              <thead>
                <tr className="bg-gray-50">
                  <th className="p-2 text-left border-b border-gray-200">機構</th>
                  <th className="p-2 text-left border-b border-gray-200">防ぐ攻撃</th>
                  <th className="p-2 text-left border-b border-gray-200">紐付け</th>
                  <th className="p-2 text-left border-b border-gray-200">検証側</th>
                  <th className="p-2 text-left border-b border-gray-200">RFC</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td className="p-2 border-b border-gray-100 font-mono font-medium">state</td>
                  <td className="p-2 border-b border-gray-100">CSRF（認可コード注入）</td>
                  <td className="p-2 border-b border-gray-100">認可リクエスト ↔ コールバック</td>
                  <td className="p-2 border-b border-gray-100">RP</td>
                  <td className="p-2 border-b border-gray-100">RFC 6749 §10.12</td>
                </tr>
                <tr>
                  <td className="p-2 border-b border-gray-100 font-mono font-medium">nonce</td>
                  <td className="p-2 border-b border-gray-100">ID Token リプレイ</td>
                  <td className="p-2 border-b border-gray-100">セッション ↔ ID Token</td>
                  <td className="p-2 border-b border-gray-100">RP</td>
                  <td className="p-2 border-b border-gray-100">OIDC Core §15.5.2</td>
                </tr>
                <tr>
                  <td className="p-2 font-mono font-medium">PKCE</td>
                  <td className="p-2">認可コード横取り</td>
                  <td className="p-2">認可リクエスト ↔ トークンリクエスト</td>
                  <td className="p-2 font-medium">OP</td>
                  <td className="p-2">RFC 7636</td>
                </tr>
              </tbody>
            </table>
          </div>
        </details>
      </div>

      {!config.stateEnabled && <CsrfAttackDemo />}
      {!config.pkceEnabled && <PkceAttackDemo />}
      {!config.nonceEnabled && <NonceAttackDemo />}
    </div>
  );
}

// =============================================================================
// CSRF 攻撃デモ（state 無効化時に表示）
// =============================================================================

function CsrfAttackDemo() {
  const [step, setStep] = useState(0);
  const [callbackUrl, setCallbackUrl] = useState<string | null>(null);
  const [fetchingLink, setFetchingLink] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchCsrfLink = useCallback(async () => {
    setFetchingLink(true);
    setError(null);
    try {
      const res = await fetch("/api/demo/csrf-link", { method: "POST" });
      if (!res.ok) throw new Error("攻撃用リンクの生成に失敗しました");
      const data = await res.json();
      setCallbackUrl(data.callbackUrl);
      setStep(4);
    } catch (e) {
      setError(e instanceof Error ? e.message : "エラーが発生しました");
    } finally {
      setFetchingLink(false);
    }
  }, []);

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <h3 className="text-lg font-bold mb-2">
        CSRF 攻撃デモ — state 無効化
      </h3>
      <p className="text-xs text-gray-500 mb-4">
        RFC 6749 §10.12: 攻撃者が自分の認可コードを被害者のセッションに注入する攻撃
      </p>

      <div className="space-y-4">
        {/* Step 1: 現在の状態確認 */}
        <StepBox
          number={1}
          title="準備 — 現在の状態確認"
          phase="準備"
          active={step >= 0}
        >
          <p className="text-sm text-gray-600">
            あなたは現在、通常のアカウント（testuser）で RP にログインしています。
          </p>
          {step === 0 && (
            <button
              onClick={() => setStep(1)}
              className="mt-2 px-3 py-1.5 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
            >
              確認しました
            </button>
          )}
        </StepBox>

        {/* Step 2: state 無効化の確認 */}
        {step >= 1 && (
          <StepBox number={2} title="準備 — state 無効化" phase="準備" active>
            <div className="p-2 bg-red-50 border border-red-200 rounded text-sm text-red-700">
              state を無効にしました。RP は callback 時に state を検証しません。
            </div>
            {step === 1 && (
              <button
                onClick={() => setStep(3)}
                className="mt-2 px-3 py-1.5 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
              >
                次へ
              </button>
            )}
          </StepBox>
        )}

        {/* Step 3: 攻撃者の認可コードを取得 */}
        {step >= 3 && (
          <StepBox number={3} title="攻撃 — 攻撃者の認可コードを取得" phase="攻撃" active>
            <p className="text-sm text-gray-600">
              攻撃者（attacker@example.com）が OP で自分のアカウントにログインし、
              認可コード付き callback URL を取得します。
            </p>
            {step === 3 && (
              <button
                onClick={fetchCsrfLink}
                disabled={fetchingLink}
                className="mt-2 px-3 py-1.5 bg-red-600 text-white text-sm rounded hover:bg-red-700 disabled:opacity-50"
              >
                {fetchingLink ? "取得中..." : "攻撃者の認可コードを取得"}
              </button>
            )}
            {error && (
              <p className="mt-2 text-sm text-red-600">{error}</p>
            )}
          </StepBox>
        )}

        {/* Step 4: callback URL の表示 + リンクを踏む */}
        {step >= 4 && callbackUrl && (
          <StepBox number={4} title="攻撃 — 被害者としてリンクを踏む" phase="攻撃" active>
            <p className="text-sm text-gray-600 mb-2">
              攻撃者はこの URL をメールや掲示板で被害者に送りつけます。
              URL に含まれる <code className="bg-gray-100 px-1 rounded text-xs">code=...</code> が攻撃者の認可コードです。
            </p>
            <div className="p-2 bg-gray-50 rounded border border-gray-200 text-xs font-mono break-all">
              {callbackUrl}
            </div>
            <p className="text-sm text-gray-600 mt-2">
              被害者として、このリンクを踏んでみましょう。
            </p>
            <button
              onClick={() => {
                window.location.href = callbackUrl;
              }}
              className="mt-2 px-3 py-1.5 bg-red-600 text-white text-sm rounded hover:bg-red-700"
            >
              被害者としてこのリンクを踏む
            </button>
            <p className="mt-2 text-xs text-gray-400">
              ⚠ クリックすると、攻撃者のアカウントで RP にログインしてしまいます
            </p>
          </StepBox>
        )}
      </div>
    </div>
  );
}

// =============================================================================
// PKCE 攻撃デモ（PKCE 無効化時に表示）
// =============================================================================

function PkceAttackDemo() {
  const [step, setStep] = useState(0);
  const [authCode, setAuthCode] = useState<string | null>(null);
  const [stealing, setStealing] = useState(false);
  const [result, setResult] = useState<{
    success: boolean;
    userinfo?: Record<string, unknown> | null;
    error?: string;
    error_description?: string;
    message?: string;
  } | null>(null);

  // Cookie から認可コードを読み取る
  useEffect(() => {
    const cookies = document.cookie.split(";").map((c) => c.trim());
    const codeCookie = cookies.find((c) => c.startsWith("demo_last_auth_code="));
    if (codeCookie) {
      setAuthCode(codeCookie.split("=")[1]);
    }
  }, []);

  const stealCode = useCallback(async () => {
    if (!authCode) return;
    setStealing(true);
    setResult(null);
    try {
      const res = await fetch("/api/demo/steal-code", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ code: authCode }),
      });
      const data = await res.json();
      setResult(data);
      setStep(5);
    } catch {
      setResult({ success: false, error: "request_failed" });
    } finally {
      setStealing(false);
    }
  }, [authCode]);

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <h3 className="text-lg font-bold mb-2">
        PKCE 攻撃デモ — 認可コード横取り
      </h3>
      <p className="text-xs text-gray-500 mb-4">
        RFC 7636: 傍受した認可コードで code_verifier なしにトークンを取得する攻撃
      </p>

      <div className="space-y-4">
        {/* Step 1: PKCE OFF 確認 */}
        <StepBox number={1} title="準備 — PKCE 無効化の確認" phase="準備" active={step >= 0}>
          <div className="p-2 bg-red-50 border border-red-200 rounded text-sm text-red-700">
            PKCE を無効にしました。認可リクエストに code_challenge は含まれません。
          </div>
          {step === 0 && (
            <button
              onClick={() => setStep(1)}
              className="mt-2 px-3 py-1.5 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
            >
              確認しました
            </button>
          )}
        </StepBox>

        {/* Step 2: ログインを促す */}
        {step >= 1 && (
          <StepBox number={2} title="準備 — PKCE なしでログイン" phase="準備" active>
            <p className="text-sm text-gray-600">
              PKCE を無効にした状態でログインしてください。ログイン後、ダッシュボードのこのパネルに戻ってきます。
            </p>
            {!authCode && (
              <p className="mt-2 text-xs text-amber-600">
                まだ認可コードが取得されていません。ログインしてからこのページに戻ってください。
              </p>
            )}
            {authCode && step === 1 && (
              <button
                onClick={() => setStep(3)}
                className="mt-2 px-3 py-1.5 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
              >
                ログイン済み — 次へ
              </button>
            )}
          </StepBox>
        )}

        {/* Step 3: 認可コード表示 */}
        {step >= 3 && authCode && (
          <StepBox number={3} title="攻撃 — 認可コードの傍受" phase="攻撃" active>
            <p className="text-sm text-gray-600 mb-2">
              攻撃者はブラウザ拡張やログ等からこの認可コードを入手しました。
            </p>
            <div className="p-2 bg-gray-50 rounded border border-gray-200 text-xs font-mono break-all">
              {authCode}
            </div>
            {step === 3 && (
              <button
                onClick={() => setStep(4)}
                className="mt-2 px-3 py-1.5 bg-red-600 text-white text-sm rounded hover:bg-red-700"
              >
                次へ
              </button>
            )}
          </StepBox>
        )}

        {/* Step 4: トークン取得の試行 */}
        {step >= 4 && (
          <StepBox number={4} title="攻撃 — code_verifier なしでトークン取得" phase="攻撃" active>
            <p className="text-sm text-gray-600">
              攻撃者はこの認可コードを使って、code_verifier なしで OP のトークンエンドポイントにリクエストします。
            </p>
            {step === 4 && (
              <button
                onClick={stealCode}
                disabled={stealing}
                className="mt-2 px-3 py-1.5 bg-red-600 text-white text-sm rounded hover:bg-red-700 disabled:opacity-50"
              >
                {stealing ? "リクエスト中..." : "攻撃者としてトークンを取得"}
              </button>
            )}
          </StepBox>
        )}

        {/* Step 5: 結果表示 */}
        {step >= 5 && result && (
          <StepBox
            number={5}
            title={result.success ? "被害 — トークン取得成功" : "防御 — トークン取得失敗"}
            phase={result.success ? "被害" : "防御"}
            active
          >
            {result.success ? (
              <>
                <div className="p-2 bg-red-50 border border-red-200 rounded text-sm text-red-700 mb-2">
                  ⚠ 攻撃成功: 攻撃者は code_verifier なしでトークンを取得できました
                </div>
                {result.userinfo && (
                  <div className="p-2 bg-gray-50 rounded border border-gray-200">
                    <p className="text-xs text-gray-500 mb-1">攻撃者が取得した被害者の情報:</p>
                    <pre className="text-xs font-mono whitespace-pre-wrap">
                      {JSON.stringify(result.userinfo, null, 2)}
                    </pre>
                  </div>
                )}
              </>
            ) : (
              <div className="p-2 bg-green-50 border border-green-200 rounded text-sm text-green-700">
                防御成功: OP が code_verifier の検証でリクエストを拒否しました
                {result.error && (
                  <span className="block mt-1 font-mono text-xs">{result.error}: {result.error_description}</span>
                )}
              </div>
            )}
          </StepBox>
        )}
      </div>
    </div>
  );
}

// =============================================================================
// nonce 攻撃デモ（nonce 無効化時に表示）
// =============================================================================

function NonceAttackDemo() {
  const [step, setStep] = useState(0);
  const [idToken, setIdToken] = useState<string | null>(null);
  const [idTokenClaims, setIdTokenClaims] = useState<Record<string, unknown> | null>(null);
  const [replaying, setReplaying] = useState(false);
  const [result, setResult] = useState<{
    accepted: boolean;
    reason?: string;
    message?: string;
    claims?: Record<string, unknown>;
    idTokenNonce?: string;
  } | null>(null);

  // Cookie から ID Token を読み取る（ダッシュボードの TokenViewer と同じソース）
  useEffect(() => {
    // セッション情報から ID Token を取得するために dashboard API を叩く
    fetch("/api/demo/config")
      .then(() => {
        // ID Token は Cookie には保存されていないので、ページの DOM から取得する代わりに
        // dashboard のデータを使う。ここでは簡易的に window から取得
      })
      .catch(() => {});
  }, []);

  const handleSetIdToken = useCallback((e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const token = e.target.value.trim();
    setIdToken(token || null);
    if (token) {
      try {
        const parts = token.split(".");
        if (parts.length === 3) {
          const payload = JSON.parse(atob(parts[1].replace(/-/g, "+").replace(/_/g, "/")));
          setIdTokenClaims(payload);
        }
      } catch {
        setIdTokenClaims(null);
      }
    }
  }, []);

  const replayIdToken = useCallback(async (nonceCheckEnabled: boolean) => {
    if (!idToken) return;
    setReplaying(true);
    setResult(null);
    try {
      const res = await fetch("/api/demo/replay-idtoken", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ idToken, nonceCheckEnabled }),
      });
      const data = await res.json();
      setResult(data);
      setStep(nonceCheckEnabled ? 6 : 5);
    } catch {
      setResult({ accepted: false, reason: "request_failed", message: "リクエストに失敗しました" });
    } finally {
      setReplaying(false);
    }
  }, [idToken]);

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <h3 className="text-lg font-bold mb-2">
        nonce 攻撃デモ — ID Token リプレイ
      </h3>
      <p className="text-xs text-gray-500 mb-2">
        OIDC Core §15.5.2: 漏洩した ID Token を別セッションで再利用するなりすまし攻撃
      </p>
      <div className="mb-4 p-2 bg-blue-50 border border-blue-200 rounded text-xs text-blue-700">
        <p className="font-medium mb-1">リプレイ攻撃の 2 パターン:</p>
        <p>1. なりすまし: 盗んだ ID Token で被害者になりすます（本デモ）</p>
        <p>2. 注入: 自分の ID Token を被害者に使わせる（state の CSRF と同方向）</p>
      </div>

      <div className="space-y-4">
        {/* Step 1: nonce OFF 確認 */}
        <StepBox number={1} title="準備 — nonce 無効化の確認" phase="準備" active={step >= 0}>
          <div className="p-2 bg-red-50 border border-red-200 rounded text-sm text-red-700">
            nonce を無効にしました。RP は ID Token の nonce を検証しません。
          </div>
          {step === 0 && (
            <button
              onClick={() => setStep(1)}
              className="mt-2 px-3 py-1.5 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
            >
              確認しました
            </button>
          )}
        </StepBox>

        {/* Step 2: ID Token 入力 */}
        {step >= 1 && (
          <StepBox number={2} title="準備 — ID Token の確認" phase="準備" active>
            <p className="text-sm text-gray-600 mb-2">
              ダッシュボード上部の「ID トークン」パネルから JWT 文字列をコピーして貼り付けてください。
              これがログ漏洩等で攻撃者に渡った ID Token に相当します。
            </p>
            <textarea
              className="w-full p-2 border border-gray-300 rounded text-xs font-mono h-20 resize-y"
              placeholder="eyJhbGciOiJSUzI1NiIs..."
              onChange={handleSetIdToken}
            />
            {idTokenClaims && (
              <div className="mt-2 p-2 bg-gray-50 rounded border border-gray-200">
                <p className="text-xs text-gray-500 mb-1">ID Token ペイロード:</p>
                <div className="text-xs font-mono space-y-0.5">
                  <p>sub: {String(idTokenClaims.sub ?? "-")}</p>
                  <p>email: {String(idTokenClaims.email ?? "-")}</p>
                  <p>name: {String(idTokenClaims.name ?? "-")}</p>
                  <p className={idTokenClaims.nonce ? "text-amber-600 font-medium" : ""}>
                    nonce: {String(idTokenClaims.nonce ?? "(なし)")}
                  </p>
                </div>
              </div>
            )}
            {step === 1 && idToken && (
              <button
                onClick={() => setStep(3)}
                className="mt-2 px-3 py-1.5 bg-blue-600 text-white text-sm rounded hover:bg-blue-700"
              >
                次へ
              </button>
            )}
          </StepBox>
        )}

        {/* Step 3: 漏洩の想定 */}
        {step >= 3 && (
          <StepBox number={3} title="攻撃 — ID Token の漏洩" phase="攻撃" active>
            <p className="text-sm text-gray-600">
              攻撃者は RP のログからこの ID Token を入手しました。
              この ID Token を使って被害者になりすまします。
            </p>
            {step === 3 && (
              <button
                onClick={() => setStep(4)}
                className="mt-2 px-3 py-1.5 bg-red-600 text-white text-sm rounded hover:bg-red-700"
              >
                次へ
              </button>
            )}
          </StepBox>
        )}

        {/* Step 4: リプレイ実行（nonce 検証なし） */}
        {step >= 4 && (
          <StepBox number={4} title="攻撃 — ID Token でログインを試みる" phase="攻撃" active>
            <p className="text-sm text-gray-600">
              攻撃者としてこの ID Token を RP に送信し、被害者になりすまします。
              nonce が無効なので、RP は別セッションの ID Token も受け入れてしまいます。
            </p>
            {step === 4 && (
              <button
                onClick={() => replayIdToken(false)}
                disabled={replaying}
                className="mt-2 px-3 py-1.5 bg-red-600 text-white text-sm rounded hover:bg-red-700 disabled:opacity-50"
              >
                {replaying ? "送信中..." : "攻撃者として ID Token を送信"}
              </button>
            )}
          </StepBox>
        )}

        {/* Step 5: 結果表示（nonce なし） */}
        {step >= 5 && result && !result.accepted === false && (
          <StepBox
            number={5}
            title={result.accepted ? "被害 — なりすまし成功" : "結果"}
            phase={result.accepted ? "被害" : "防御"}
            active
          >
            {result.accepted ? (
              <>
                <div className="p-2 bg-red-50 border border-red-200 rounded text-sm text-red-700 mb-2">
                  ⚠ 攻撃成功: 被害者の ID Token が別セッションで受け入れられました
                </div>
                {result.claims && (
                  <div className="p-2 bg-gray-50 rounded border border-gray-200">
                    <p className="text-xs text-gray-500 mb-1">攻撃者が取得した被害者の情報:</p>
                    <pre className="text-xs font-mono whitespace-pre-wrap">
                      {JSON.stringify(result.claims, null, 2)}
                    </pre>
                  </div>
                )}
                <button
                  onClick={() => {
                    setResult(null);
                    setStep(6);
                  }}
                  className="mt-2 px-3 py-1.5 bg-green-600 text-white text-sm rounded hover:bg-green-700"
                >
                  防御を確認する（nonce 検証あり）
                </button>
              </>
            ) : (
              <div className="p-2 bg-gray-50 rounded text-sm text-gray-600">
                {result.message}
              </div>
            )}
          </StepBox>
        )}

        {/* Step 6: 防御確認（nonce 検証あり） */}
        {step >= 6 && (
          <StepBox number={6} title="防御 — nonce 検証ありで再試行" phase="防御" active>
            <p className="text-sm text-gray-600">
              同じ ID Token を nonce 検証ありで送信します。
              nonce がセッションと ID Token を 1:1 で紐付けているため、別セッションでのリプレイが検出されます。
            </p>
            {!result && (
              <button
                onClick={() => replayIdToken(true)}
                disabled={replaying}
                className="mt-2 px-3 py-1.5 bg-green-600 text-white text-sm rounded hover:bg-green-700 disabled:opacity-50"
              >
                {replaying ? "送信中..." : "nonce 検証ありで再試行"}
              </button>
            )}
            {result && !result.accepted && (
              <div className="mt-2 p-2 bg-green-50 border border-green-200 rounded text-sm text-green-700">
                防御成功: {result.message}
              </div>
            )}
          </StepBox>
        )}
      </div>
    </div>
  );
}

// =============================================================================
// ステップ表示用コンポーネント
// =============================================================================

function StepBox({
  number,
  title,
  phase,
  active,
  children,
}: {
  number: number;
  title: string;
  phase: "準備" | "攻撃" | "被害" | "防御";
  active: boolean;
  children: React.ReactNode;
}) {
  const phaseColors = {
    準備: "bg-blue-100 text-blue-700",
    攻撃: "bg-red-100 text-red-700",
    被害: "bg-orange-100 text-orange-700",
    防御: "bg-green-100 text-green-700",
  };

  return (
    <div className={`p-4 rounded border ${active ? "border-gray-300 bg-white" : "border-gray-100 bg-gray-50 opacity-50"}`}>
      <div className="flex items-center gap-2 mb-2">
        <span className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-gray-200 text-xs font-bold">
          {number}
        </span>
        <span className={`text-xs px-1.5 py-0.5 rounded font-medium ${phaseColors[phase]}`}>
          {phase}
        </span>
        <span className="text-sm font-medium">{title}</span>
      </div>
      {children}
    </div>
  );
}
