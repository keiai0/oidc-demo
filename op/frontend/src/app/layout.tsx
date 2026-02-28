import "./globals.css";

export const metadata = {
  title: "OP 管理画面",
  description: "OpenID Provider 管理 UI",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="ja">
      <body className="bg-gray-50 text-gray-900">{children}</body>
    </html>
  );
}
