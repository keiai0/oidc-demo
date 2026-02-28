"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useManagementAuth } from "@/lib/management-auth";
import { routes } from "@/lib/routes";

const navItems = [
  { href: routes.management.root, label: "ダッシュボード" },
  { href: routes.management.tenants, label: "テナント" },
  { href: routes.management.keys, label: "署名鍵" },
  { href: routes.management.incidents, label: "インシデント" },
];

export function Sidebar() {
  const pathname = usePathname();
  const { user, logout } = useManagementAuth();

  const handleLogout = async () => {
    await logout();
    window.location.href = routes.management.login;
  };

  return (
    <aside className="w-60 shrink-0 border-r border-gray-200 bg-white flex flex-col">
      <div className="p-4 border-b border-gray-200">
        <h1 className="text-lg font-bold text-gray-900">OP 管理画面</h1>
      </div>
      <nav className="p-2 flex flex-col gap-1 flex-1">
        {navItems.map((item) => {
          const active =
            item.href === routes.management.root
              ? pathname === routes.management.root
              : pathname.startsWith(item.href);
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`block px-3 py-2 rounded text-sm ${
                active
                  ? "bg-blue-50 text-blue-700 font-medium"
                  : "text-gray-700 hover:bg-gray-100"
              }`}
            >
              {item.label}
            </Link>
          );
        })}
      </nav>
      <div className="p-4 border-t border-gray-200">
        {user && (
          <p className="text-xs text-gray-500 mb-2 truncate">{user.name}</p>
        )}
        <button
          onClick={handleLogout}
          className="w-full px-3 py-2 text-sm text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded"
        >
          ログアウト
        </button>
      </div>
    </aside>
  );
}
