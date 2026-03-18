"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

/** OP がサポートするクレーム一覧 */
const AVAILABLE_CLAIMS = {
  id_token: ["sub", "name", "email", "email_verified", "updated_at", "auth_time"],
  userinfo: ["sub", "name", "email", "email_verified", "updated_at"],
} as const;

interface ClaimEntry {
  enabled: boolean;
  essential: boolean;
}

type ClaimsState = Record<string, Record<string, ClaimEntry>>;

function buildInitialState(): ClaimsState {
  const state: ClaimsState = {};
  for (const [target, claims] of Object.entries(AVAILABLE_CLAIMS)) {
    state[target] = {};
    for (const claim of claims) {
      state[target][claim] = { enabled: false, essential: false };
    }
  }
  return state;
}

function buildClaimsJson(state: ClaimsState): string | null {
  const result: Record<string, Record<string, { essential?: boolean } | null>> = {};
  let hasAnyClaim = false;

  for (const [target, claims] of Object.entries(state)) {
    const targetClaims: Record<string, { essential?: boolean } | null> = {};
    for (const [name, entry] of Object.entries(claims)) {
      if (entry.enabled) {
        hasAnyClaim = true;
        targetClaims[name] = entry.essential ? { essential: true } : null;
      }
    }
    if (Object.keys(targetClaims).length > 0) {
      result[target] = targetClaims;
    }
  }

  return hasAnyClaim ? JSON.stringify(result) : null;
}

export function ClaimsConfig() {
  const router = useRouter();
  const [expanded, setExpanded] = useState(false);
  const [state, setState] = useState<ClaimsState>(buildInitialState);

  const toggleClaim = (target: string, claim: string) => {
    setState((prev) => ({
      ...prev,
      [target]: {
        ...prev[target],
        [claim]: {
          ...prev[target][claim],
          enabled: !prev[target][claim].enabled,
          essential: !prev[target][claim].enabled ? prev[target][claim].essential : false,
        },
      },
    }));
  };

  const toggleEssential = (target: string, claim: string) => {
    setState((prev) => ({
      ...prev,
      [target]: {
        ...prev[target],
        [claim]: {
          ...prev[target][claim],
          essential: !prev[target][claim].essential,
        },
      },
    }));
  };

  const claimsJson = buildClaimsJson(state);

  const handleLogin = () => {
    const url = claimsJson
      ? `/api/auth/login?claims=${encodeURIComponent(claimsJson)}`
      : "/api/auth/login";
    router.push(url);
  };

  return (
    <div>
      <button
        onClick={handleLogin}
        className="block w-full py-3 px-4 bg-blue-600 text-white text-center rounded-lg hover:bg-blue-700 transition-colors font-medium cursor-pointer"
      >
        OP でログイン
      </button>

      <button
        onClick={() => setExpanded(!expanded)}
        className="mt-4 w-full text-left text-sm text-gray-500 hover:text-gray-700 flex items-center gap-1 cursor-pointer"
      >
        <span className={`transition-transform ${expanded ? "rotate-90" : ""}`}>
          ▶
        </span>
        Claims パラメータ設定（OIDC Core Section 5.5）
      </button>

      {expanded && (
        <div className="mt-3 p-4 bg-gray-50 rounded-lg border border-gray-200 space-y-4">
          {Object.entries(AVAILABLE_CLAIMS).map(([target, claims]) => (
            <div key={target}>
              <h3 className="text-xs font-semibold text-gray-600 mb-2 uppercase">
                {target}
              </h3>
              <div className="space-y-1">
                {claims.map((claim) => (
                  <div key={claim} className="flex items-center gap-3 text-sm">
                    <label className="flex items-center gap-1.5 min-w-[140px] cursor-pointer">
                      <input
                        type="checkbox"
                        checked={state[target][claim].enabled}
                        onChange={() => toggleClaim(target, claim)}
                        className="rounded"
                      />
                      <code className="text-xs bg-gray-100 px-1 rounded">
                        {claim}
                      </code>
                    </label>
                    {state[target][claim].enabled && (
                      <label className="flex items-center gap-1 text-xs text-gray-500 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={state[target][claim].essential}
                          onChange={() => toggleEssential(target, claim)}
                          className="rounded"
                        />
                        essential
                      </label>
                    )}
                  </div>
                ))}
              </div>
            </div>
          ))}

          {claimsJson && (
            <div>
              <h3 className="text-xs font-semibold text-gray-600 mb-1">
                生成される JSON
              </h3>
              <pre className="bg-white rounded p-2 text-xs font-mono overflow-x-auto border border-gray-200">
                {JSON.stringify(JSON.parse(claimsJson), null, 2)}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
