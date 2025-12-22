import "./globals.css";
import Script from "next/script";

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        {/* DNS Prefetch & Preconnect for Analytics */}
        <link rel="dns-prefetch" href="https://www.googletagmanager.com" />
        <link rel="dns-prefetch" href="https://www.google-analytics.com" />

        {/* Preload critical fonts */}
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

        {/* Preload logo */}
        <link rel="preload" href="/logo.svg" as="image" type="image/svg+xml" />
      </head>

      <body style={{ fontFamily: "Jost, sans-serif" }}>
        {children}
        <Script
          src="https://www.googletagmanager.com/gtag/js?id=G-GE01THYR9X"
          strategy="lazyOnload"
        />
        <Script id="google-analytics" strategy="lazyOnload">
          {`
            window.dataLayer = window.dataLayer || [];
            function gtag(){dataLayer.push(arguments);}
            gtag('js', new Date());
            gtag('config', 'G-GE01THYR9X');
          `}
        </Script>
      </body>
    </html>
  );
}
