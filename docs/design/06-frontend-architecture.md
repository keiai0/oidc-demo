# 06. フロントエンドアーキテクチャ

## 概要

OP Frontend（管理画面 + ログイン UI）のアーキテクチャ設計。
小〜中規模の管理画面を対象とし、過剰な抽象化を避けつつ、コードの重複と型の不整合を解消する。

## 技術スタック

| ライブラリ | バージョン | 用途 |
|---|---|---|
| Next.js | 16 (App Router) | フレームワーク。`output: "export"` で静的出力 |
| React | 19 | UI ライブラリ |
| TypeScript | 5.7 (strict) | 型安全性 |
| Tailwind CSS | v4 | スタイリング |
| @tanstack/react-query | 5 | サーバー状態管理・キャッシュ・自動リフェッチ |
| react-hook-form | 7 | フォーム管理（非制御コンポーネント） |
| zod | 3 | スキーマバリデーション |
| @hookform/resolvers | 3 | zod + react-hook-form 連携 |
| pnpm | 9 | パッケージマネージャ |

### 選定理由

- **TanStack Query**: 全ページで繰り返されていた `useState` + `useEffect` + `loading/error/data` パターンを一掃する。ミューテーション後のキャッシュ無効化により手動リフェッチが不要になる。
- **react-hook-form + zod**: 3 つ以上のフォームページで異なるフォームパターンが混在していた問題を解消。zod スキーマでバックエンドのバリデーション制約をフロントでも再現する。

## ディレクトリ構成

```
op/frontend/src/
├── app/                          # Next.js App Router ページ
│   ├── layout.tsx                # ルートレイアウト（html/body, globals.css import）
│   ├── page.tsx                  # トップページ
│   ├── globals.css               # Tailwind v4 import
│   ├── login/
│   │   └── page.tsx              # エンドユーザーログイン（OIDC 認証フロー用）
│   └── management/
│       ├── layout.tsx            # Providers のみ（QueryClient, AuthProvider）
│       ├── login/
│       │   └── page.tsx          # 管理ログインフォーム（AuthGuard なし）
│       └── (authed)/             # Route Group: 認証必須ページ
│           ├── layout.tsx        # AuthGuard + Sidebar
│           ├── page.tsx          # ダッシュボード
│           ├── tenants/
│           │   ├── page.tsx      # テナント一覧
│           │   ├── new/page.tsx  # テナント作成
│           │   └── detail/
│           │       ├── page.tsx  # テナント詳細・編集
│           │       └── clients/
│           │           ├── page.tsx  # クライアント一覧
│           │           └── new/page.tsx  # クライアント作成
│           ├── clients/
│           │   └── detail/page.tsx   # クライアント詳細（URI管理・シークレット・削除）
│           ├── keys/
│           │   └── page.tsx          # 署名鍵管理
│           └── incidents/
│               └── page.tsx          # インシデント対応（トークン失効）
│
├── components/                   # 再利用可能コンポーネント
│   ├── ui/                       # 汎用 UI プリミティブ
│   │   ├── alert.tsx             # エラー・成功・警告メッセージ
│   │   ├── badge.tsx             # ステータスバッジ（active/inactive）
│   │   ├── card.tsx              # カードコンテナ
│   │   ├── data-table.tsx        # テーブル（ヘッダー定義 + 空状態）
│   │   ├── loading.tsx           # ローディング表示
│   │   └── page-header.tsx       # ページタイトル + アクションボタン
│   ├── layout/                   # レイアウト系
│   │   ├── sidebar.tsx           # ナビゲーションサイドバー
│   │   └── auth-guard.tsx        # 認証チェック + リダイレクト
│   └── form/                     # フォーム系
│       ├── form-field.tsx        # ラベル + 入力 + エラーメッセージ
│       └── uri-list-field.tsx    # 動的 URI リスト（追加/削除）
│
├── lib/                          # 非 React ユーティリティ
│   ├── api/                      # API クライアント
│   │   ├── client.ts             # fetch ラッパー + ApiRequestError
│   │   ├── tenants.ts            # テナント API
│   │   ├── clients.ts            # クライアント API
│   │   ├── keys.ts               # 署名鍵 API
│   │   ├── incidents.ts          # インシデント API
│   │   └── auth.ts               # 認証 API (login/logout/me)
│   ├── query/                    # TanStack Query 設定
│   │   ├── query-client.ts       # QueryClient 生成
│   │   └── query-keys.ts         # Query key ファクトリ
│   ├── management-auth.tsx       # 認証 Context Provider
│   └── routes.ts                 # ルート定数
│
├── schemas/                      # Zod バリデーションスキーマ
│   ├── tenant.ts                 # テナント作成・更新スキーマ
│   ├── client.ts                 # クライアント作成スキーマ
│   └── incident.ts               # インシデント操作スキーマ
│
└── types/                        # 共有 TypeScript 型定義
    ├── api.ts                    # ApiError, ListResponse<T>
    ├── tenant.ts                 # Tenant
    ├── client.ts                 # Client, ClientDetail, RedirectURI, etc.
    ├── key.ts                    # SignKey
    ├── incident.ts               # RevokeResponse
    ├── auth.ts                   # AdminUser
    └── index.ts                  # barrel export
```

### 配置ルール

| ディレクトリ | 責務 | 配置基準 |
|---|---|---|
| `types/` | API レスポンスの TypeScript 型 | バックエンドの response struct と 1:1 対応 |
| `schemas/` | Zod バリデーションスキーマ + `z.infer` 由来の入力型 | フォームで使う作成・更新リクエストの検証。infer 型はスキーマと同じファイルに export |
| `lib/api/` | 型付き API 関数 | 1 ハンドラファイル = 1 API モジュール |
| `lib/query/` | TanStack Query 設定 | query-client.ts + query-keys.ts のみ |
| `components/ui/` | 汎用 UI | 3 箇所以上で使われるパターン |
| `components/form/` | フォーム UI | react-hook-form と組み合わせて使う |
| `components/layout/` | レイアウト系 | Sidebar, AuthGuard |
| `app/` | ページ | hooks + components を組み合わせるだけ |

## 設計原則

### 1. ページは薄く

ページコンポーネントは「データ取得（useQuery）」「ミューテーション（useMutation）」「UIコンポーネント」を組み合わせるだけの薄いレイヤーとする。ビジネスロジックは API クライアント・スキーマ・コンポーネントに分散させる。

```tsx
// 理想的なページ構造
export default function TenantsPage() {
  const { data, isLoading, error } = useQuery({
    queryKey: queryKeys.tenants.list(),
    queryFn: () => tenantsApi.list(),
  });

  if (isLoading) return <Loading />;
  if (error) return <Alert variant="error">{getErrorMessage(error)}</Alert>;

  return (
    <>
      <PageHeader title="Tenants" action={<Link href={routes.management.tenants.new}>Create</Link>} />
      <DataTable columns={columns} data={data.data} rowKey={(t) => t.id} />
    </>
  );
}
```

### 2. 型はバックエンドに合わせる

`types/` の型定義はバックエンドの Go response struct を正確に反映する。フロントエンド独自の型変換は行わない。

| バックエンド (Go) | フロントエンド (TypeScript) |
|---|---|
| `tenantResponse` | `Tenant` |
| `clientResponse` | `Client` |
| `clientDetailResponse` | `ClientDetail` |
| `keyResponse` | `SignKey` |
| `revokeResponse` | `RevokeResponse` |
| `ListResponse[T]` | `ListResponse<T>` |
| `ErrorResponse` | `ApiError` |

### 3. 過剰な抽象化を避ける

- 2 箇所以下でしか使われないパターンはコンポーネント化しない
- ジェネリックすぎるコンポーネント（任意の props を受ける万能コンポーネント）は作らない
- カスタム hooks は TanStack Query が担う領域と重複しない範囲でのみ作る

## API クライアント設計

### fetch ラッパー (`lib/api/client.ts`)

```typescript
export class ApiRequestError extends Error {
  constructor(
    public readonly code: string,
    public readonly description: string,
    public readonly status: number,
  ) {
    super(description);
    this.name = "ApiRequestError";
  }
}

export async function managementFetch<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    credentials: "include",  // Cookie ベース認証
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });

  if (!res.ok) {
    // ApiRequestError として throw
  }

  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}
```

- 認証は Cookie ベース（`op_admin_session`）。`credentials: "include"` で自動送信。
- API Key 方式は使用しない（管理者セッション Cookie を使用）。
- `ApiRequestError` は `instanceof` チェック可能な Error サブクラス。`status` フィールドで 401 判定が可能。

### API モジュール (`lib/api/{resource}.ts`)

```typescript
// 例: tenantsApi
export const tenantsApi = {
  list: (params?) => managementFetch<ListResponse<Tenant>>(`/management/v1/tenants${qs}`),
  get: (id: string) => managementFetch<Tenant>(`/management/v1/tenants/${id}`),
  create: (body) => managementFetch<Tenant>("/management/v1/tenants", { method: "POST", body: ... }),
  update: (id, body) => managementFetch<Tenant>(`/management/v1/tenants/${id}`, { method: "PUT", body: ... }),
};
```

- 1 リソース = 1 ファイル（バックエンドの handler ファイルと対応）
- 純粋なオブジェクトリテラル（クラスではない）
- エンドポイントパスはこのレイヤーに集約（ページに散在させない）

## TanStack Query 設計

### QueryClient (`lib/query/query-client.ts`)

```typescript
export function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,  // 30 秒
        retry: (failureCount, error) => {
          if (error instanceof ApiRequestError && [401, 403, 404].includes(error.status)) {
            return false;
          }
          return failureCount < 2;
        },
      },
    },
  });
}
```

### Query Key ファクトリ (`lib/query/query-keys.ts`)

```typescript
export const queryKeys = {
  tenants: {
    all: ["tenants"] as const,
    list: (params?) => ["tenants", "list", params] as const,
    detail: (id: string) => ["tenants", "detail", id] as const,
  },
  clients: {
    all: ["clients"] as const,
    listByTenant: (tenantId: string) => ["clients", "list", { tenantId }] as const,
    detail: (id: string) => ["clients", "detail", id] as const,
  },
  keys: {
    all: ["keys"] as const,
    list: () => ["keys", "list"] as const,
  },
  dashboard: {
    stats: () => ["dashboard", "stats"] as const,
  },
};
```

- ミューテーション成功後に `invalidateQueries({ queryKey: queryKeys.tenants.all })` でキャッシュを無効化する。

## フォーム設計

### Zod スキーマ (`schemas/{resource}.ts`)

バックエンドのバリデーション制約を再現する:

```typescript
// schemas/tenant.ts
export const createTenantSchema = z.object({
  code: z.string()
    .min(3, "3文字以上")
    .max(63, "63文字以下")
    .regex(/^[a-z0-9][a-z0-9-]*[a-z0-9]$/, "小文字英数字とハイフン。先頭末尾は英数字"),
  name: z.string().min(1, "必須").max(255, "255文字以下"),
  session_lifetime: z.coerce.number().positive("正の整数").optional(),
  // ...
});
```

### react-hook-form 使用パターン

```tsx
const { register, handleSubmit, formState: { errors } } = useForm<CreateTenantInput>({
  resolver: zodResolver(createTenantSchema),
  defaultValues: { access_token_lifetime: 3600, ... },
});

const mutation = useMutation({
  mutationFn: tenantsApi.create,
  onSuccess: (tenant) => {
    queryClient.invalidateQueries({ queryKey: queryKeys.tenants.all });
    // 遷移
  },
});

return (
  <form onSubmit={handleSubmit((values) => mutation.mutate(values))}>
    <FormField label="Code" error={errors.code?.message}>
      <input {...register("code")} />
    </FormField>
    {mutation.error && <Alert variant="error">{getErrorMessage(mutation.error)}</Alert>}
    <button type="submit" disabled={mutation.isPending}>作成</button>
  </form>
);
```

## 認証設計

### 管理者認証フロー

1. `/management/login` で login_id + password を入力
2. POST `/management/v1/auth/login` → バックエンドが `op_admin_session` Cookie をセット
3. 以降の API リクエストは Cookie が自動送信される（`credentials: "include"`）
4. ページ遷移時に GET `/management/v1/auth/me` でセッション有効性を確認

### AuthGuard (`components/layout/auth-guard.tsx`)

```tsx
export function AuthGuard({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useManagementAuth();

  // isLoading 中はローディング表示（競合状態の修正）
  if (isLoading) {
    return <Loading message="認証確認中..." />;
  }

  if (!isAuthenticated) {
    window.location.href = routes.management.login;
    return null;
  }

  return (
    <div className="flex h-screen">
      <Sidebar />
      <main className="flex-1 overflow-y-auto p-6">{children}</main>
    </div>
  );
}
```

### Provider 構成

Route Group `(authed)` を使い、認証の有無でレイアウトを分離する:

```
management/layout.tsx          → QueryClientProvider + ManagementAuthProvider のみ
management/(authed)/layout.tsx → AuthGuard + Sidebar
management/login/page.tsx      → AuthGuard なしでアクセス可能
```

```tsx
// management/layout.tsx — Providers のみ
<QueryClientProvider client={queryClient}>
  <ManagementAuthProvider>
    {children}
  </ManagementAuthProvider>
</QueryClientProvider>
```

```tsx
// management/(authed)/layout.tsx — 認証必須レイアウト
<AuthGuard>{children}</AuthGuard>
```

`ManagementAuthProvider` は `management/layout.tsx` で適用されるため、login ページからも `useManagementAuth()` の `login()` を利用できる。ErrorBoundary も `management/(authed)/layout.tsx` の AuthGuard 内に配置する。

## ルーティング

静的エクスポート（`output: "export"`）の制約により、動的セグメント（`/tenants/[id]`）は使用せず、query params（`?id=xxx`）方式を維持する。

ハードコードされた URL 文字列を排除するため、ルート定数を導入:

```typescript
// lib/routes.ts
export const routes = {
  login: "/login",
  management: {
    root: "/management",
    login: "/management/login",
    tenants: {
      list: "/management/tenants",
      new: "/management/tenants/new",
      detail: (id: string) => `/management/tenants/detail?id=${id}`,
      clients: {
        list: (tenantId: string) => `/management/tenants/detail/clients?tenant_id=${tenantId}`,
        new: (tenantId: string) => `/management/tenants/detail/clients/new?tenant_id=${tenantId}`,
      },
    },
    clients: {
      detail: (id: string) => `/management/clients/detail?id=${id}`,
    },
    keys: "/management/keys",
    incidents: "/management/incidents",
  },
};
```

## エラーハンドリング

### 3 層構造

1. **TanStack Query レベル**: `useQuery` / `useMutation` の error state。各ページが `<Alert>` で表示。
2. **401 処理（後述）**: セッション切れを検知して自動的にログイン画面へリダイレクト。
3. **React Error Boundary**: 予期せぬレンダリングエラーのキャッチ。Next.js の `error.tsx` 規約は静的エクスポートでは制限があるため、class component ベースの ErrorBoundary を management layout に配置。

### 401 処理

TanStack Query v5 では queries のグローバル `onError` が廃止されている。以下の方式で対応する:

- **query**: セッション切れで API が 401 を返すと、`management-auth.tsx` の `/management/v1/auth/me` チェックも失敗し `isAuthenticated` が `false` になる。AuthGuard がログインリダイレクトを実行する。
- **mutation**: 個別の `onError` コールバックで `ApiRequestError` の `status === 401` を検知してリダイレクト。

### エラーメッセージ抽出ヘルパー (`lib/api/client.ts` に配置)

```typescript
export function getErrorMessage(error: unknown): string {
  if (error instanceof ApiRequestError) return error.description;
  if (error instanceof Error) return error.message;
  return "予期せぬエラーが発生しました";
}
```

## バックエンドとの型対応表

| バックエンド Go struct | JSON フィールド | フロントエンド型 |
|---|---|---|
| `tenantResponse` | id, code, name, session_lifetime, auth_code_lifetime, access_token_lifetime, refresh_token_lifetime, id_token_lifetime, created_at, updated_at | `Tenant` |
| `clientResponse` | id, tenant_id, client_id, name, grant_types, response_types, token_endpoint_auth_method, require_pkce, frontchannel_logout_uri?, backchannel_logout_uri?, status, created_at, updated_at | `Client` |
| `clientDetailResponse` | clientResponse + redirect_uris, post_logout_redirect_uris | `ClientDetail` |
| `clientCreateResponse` | clientResponse + client_secret | `ClientCreateResponse` |
| `redirectURIResponse` | id, uri, created_at | `RedirectURI` |
| `keyResponse` | kid, algorithm, **active** (bool), created_at, rotated_at? | `SignKey` |
| `revokeResponse` | revoked.sessions, revoked.access_tokens, revoked.refresh_tokens | `RevokeResponse` |
| `ErrorResponse` | error, error_description | `ApiError` |
| `ListResponse[T]` | data, total_count | `ListResponse<T>` |

**注意**:
- Keys の一覧は `ListResponse` でラップされず、配列が直接返される
- Secret ローテーションは `{ client_id, client_secret }` を返す
- Auth login/me は `{ user: { id, login_id, name } }` を返す

## ログインページ (`/login`) の位置づけ

`/login` は OIDC 認可フロー中にエンドユーザーが認証するためのページであり、管理画面とは独立している。

- TanStack Query は**使用しない**（QueryClientProvider のスコープ外）
- API コールは `managementFetch` ではなく直接 `fetch` で `/internal/login` を呼ぶ
- react-hook-form + zod は使用する（バリデーションの統一）
- Tailwind でスタイリングする（インラインスタイルは使わない）
