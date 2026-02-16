"use client";

import { useState, useEffect, type FormEvent } from "react";

const API_URL = process.env.NEXT_PUBLIC_OP_API_URL || "http://localhost:8080";

export default function LoginPage() {
  const [loginId, setLoginId] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [tenantCode, setTenantCode] = useState("");
  const [redirectAfterLogin, setRedirectAfterLogin] = useState("");

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    setTenantCode(params.get("tenant_code") || "demo");
    // 認可リクエストの全クエリパラメータを保持（ログイン後にリダイレクト）
    setRedirectAfterLogin(params.get("redirect_after_login") || "");
  }, []);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const res = await fetch(`${API_URL}/internal/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          tenant_code: tenantCode,
          login_id: loginId,
          password: password,
        }),
      });

      if (!res.ok) {
        const data = await res.json();
        setError(
          data.error === "invalid_credentials"
            ? "ログインIDまたはパスワードが正しくありません"
            : "ログインに失敗しました"
        );
        return;
      }

      // ログイン成功：認可エンドポイントに戻る
      if (redirectAfterLogin) {
        window.location.href = redirectAfterLogin;
      }
    } catch {
      setError("サーバーに接続できません");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={styles.container}>
      <div style={styles.card}>
        <h1 style={styles.title}>ログイン</h1>
        {error && <div style={styles.error}>{error}</div>}
        <form onSubmit={handleSubmit}>
          <div style={styles.field}>
            <label htmlFor="loginId" style={styles.label}>
              ログインID
            </label>
            <input
              id="loginId"
              type="text"
              value={loginId}
              onChange={(e) => setLoginId(e.target.value)}
              required
              autoFocus
              style={styles.input}
            />
          </div>
          <div style={styles.field}>
            <label htmlFor="password" style={styles.label}>
              パスワード
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              style={styles.input}
            />
          </div>
          <button type="submit" disabled={loading} style={styles.button}>
            {loading ? "ログイン中..." : "ログイン"}
          </button>
        </form>
      </div>
    </div>
  );
}

const styles: Record<string, React.CSSProperties> = {
  container: {
    minHeight: "100vh",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: "#f5f5f5",
    fontFamily: "system-ui, sans-serif",
  },
  card: {
    backgroundColor: "#fff",
    padding: "2rem",
    borderRadius: "8px",
    boxShadow: "0 2px 8px rgba(0, 0, 0, 0.1)",
    width: "100%",
    maxWidth: "400px",
  },
  title: {
    fontSize: "1.5rem",
    fontWeight: 600,
    textAlign: "center" as const,
    marginBottom: "1.5rem",
    color: "#333",
  },
  error: {
    backgroundColor: "#fef2f2",
    color: "#dc2626",
    padding: "0.75rem",
    borderRadius: "4px",
    marginBottom: "1rem",
    fontSize: "0.875rem",
  },
  field: {
    marginBottom: "1rem",
  },
  label: {
    display: "block",
    marginBottom: "0.25rem",
    fontSize: "0.875rem",
    fontWeight: 500,
    color: "#555",
  },
  input: {
    width: "100%",
    padding: "0.5rem 0.75rem",
    border: "1px solid #ddd",
    borderRadius: "4px",
    fontSize: "1rem",
    boxSizing: "border-box" as const,
  },
  button: {
    width: "100%",
    padding: "0.75rem",
    backgroundColor: "#2563eb",
    color: "#fff",
    border: "none",
    borderRadius: "4px",
    fontSize: "1rem",
    fontWeight: 500,
    cursor: "pointer",
    marginTop: "0.5rem",
  },
};
