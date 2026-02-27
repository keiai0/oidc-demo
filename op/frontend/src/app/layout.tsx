import "./globals.css";

export const metadata = {
  title: "OP Frontend",
  description: "OpenID Provider Management UI",
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
