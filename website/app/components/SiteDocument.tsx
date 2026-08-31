import "@/app/globals.css";
import Script from "next/script";

export default function SiteDocument({
  htmlLang,
  children,
}: Readonly<{
  htmlLang: string;
  children: React.ReactNode;
}>) {
  return (
    <html lang={htmlLang} suppressHydrationWarning>
      <head>
        <link rel="dns-prefetch" href="https://rybbit.databasus.com" />

        <link
          rel="preload"
          href="/font/Jost-Regular.ttf"
          as="font"
          type="font/ttf"
          crossOrigin="anonymous"
        />
        <link
          rel="preload"
          href="/font/Jost-Bold.ttf"
          as="font"
          type="font/ttf"
          crossOrigin="anonymous"
        />

        <link rel="preload" href="/logo.svg" as="image" type="image/svg+xml" />
      </head>

      <body style={{ fontFamily: "Jost, sans-serif" }}>
        {children}
        <Script
          src="https://rybbit.databasus.com/api/script.js"
          data-site-id="dc1a72d4fa0d"
          strategy="lazyOnload"
        />
      </body>
    </html>
  );
}
