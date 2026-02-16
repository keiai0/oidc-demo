export const metadata = {
  title: "RP",
  description: "OIDC Relying Party (Demo)",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="ja">
      <body>{children}</body>
    </html>
  );
}
