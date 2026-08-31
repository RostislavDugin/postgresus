import SiteDocument from "@/app/components/SiteDocument";
import { HTML_LANG_CODES, LOCALES, type Locale } from "@/app/i18n";

export const dynamicParams = false;

export function generateStaticParams() {
  return LOCALES.map((lang) => ({ lang }));
}

export default async function LocalizedRootLayout({
  children,
  params,
}: Readonly<{
  children: React.ReactNode;
  params: Promise<{ lang: string }>;
}>) {
  const { lang } = await params;
  return (
    <SiteDocument htmlLang={HTML_LANG_CODES[lang as Locale]}>
      {children}
    </SiteDocument>
  );
}
