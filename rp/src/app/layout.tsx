import "./globals.css";

export const metadata = {
  title: "RP - OIDC Relying Party",
  description: "OIDC 動作検証用 Relying Party",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="ja">
      <body className="min-h-screen bg-gray-50 text-gray-900">{children}</body>
    </html>
  );
}
