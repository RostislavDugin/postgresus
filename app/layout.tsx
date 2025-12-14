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
      </head>

      <body style={{ fontFamily: "Jost, sans-serif" }}>
        {children}
        <Script
          src="https://www.googletagmanager.com/gtag/js?id=G-GE01THYR9X"
          strategy="afterInteractive"
        />
        <Script id="google-analytics" strategy="afterInteractive">
          {`
            window.dataLayer = window.dataLayer || [];
            function gtag(){dataLayer.push(arguments);}
            gtag('js', new Date());
            gtag('config', 'G-GE01THYR9X');

            // Goal 1: Track 30 second engagement
            setTimeout(function() {
              gtag('event', 'engaged_30_seconds', {
                'event_category': 'engagement',
                'event_label': 'time_on_page'
              });
            }, 30000);

            // Goal 2: Track GitHub link clicks
            document.addEventListener('click', function(e) {
              var link = e.target.closest('a');
              if (link && link.href && link.href.includes('github.com')) {
                gtag('event', 'click_github', {
                  'event_category': 'outbound',
                  'event_label': link.href,
                  'transport_type': 'beacon'
                });
              }
            });
          `}
        </Script>
      </body>
    </html>
  );
}
