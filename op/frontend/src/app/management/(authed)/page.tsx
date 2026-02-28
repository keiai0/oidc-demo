"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { tenantsApi } from "@/lib/api/tenants";
import { queryKeys } from "@/lib/query/query-keys";
import { routes } from "@/lib/routes";

export default function ダッシュボードPage() {
  const { data } = useQuery({
    queryKey: queryKeys.tenants.list(1, 0),
    queryFn: () => tenantsApi.list(1, 0),
  });

  const cards = [
    {
      title: "テナント",
      href: routes.management.tenants,
      count: data?.total_count,
      description: "テナントの管理",
    },
    {
      title: "クライアント",
      href: routes.management.clients,
      description: "OAuth クライアントの管理",
    },
    {
      title: "署名鍵",
      href: routes.management.keys,
      description: "JWT 署名鍵の管理",
    },
    {
      title: "インシデント",
      href: routes.management.incidents,
      description: "緊急トークン失効",
    },
  ];

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">ダッシュボード</h1>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {cards.map((card) => (
          <Link
            key={card.href}
            href={card.href}
            className="block bg-white rounded-lg border border-gray-200 p-6 hover:shadow-md transition-shadow"
          >
            <h2 className="text-lg font-semibold text-gray-900">
              {card.title}
            </h2>
            <p className="text-sm text-gray-500 mt-1">{card.description}</p>
            {card.count !== undefined && card.count !== null && (
              <p className="text-3xl font-bold text-blue-600 mt-3">
                {card.count}
              </p>
            )}
          </Link>
        ))}
      </div>
    </div>
  );
}
