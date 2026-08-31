import SiteDocument from "@/app/components/SiteDocument";

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return <SiteDocument htmlLang="en">{children}</SiteDocument>;
}
