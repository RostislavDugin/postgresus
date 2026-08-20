import type { Metadata } from "next";
import Script from "next/script";
import PaddleInitComponent from "../components/PaddleInitComponent";
import { PADDLE_ENABLED, TELEGRAM_URL, TIERS, priceUsesPaddle } from "./tiers";

export const metadata: Metadata = {
  title: "Sponsor Databasus | Support open-source PostgreSQL backups",
  description:
    "Databasus is free forever under Apache 2.0 — no open core, no paywalled features. Sponsorship funds the maintenance, security work and new features that everyone gets for free.",
  keywords:
    "Databasus sponsorship, sponsor open source, PostgreSQL backup, open source funding, Apache 2.0, support Databasus, fund a feature",
  robots: "index, follow",
  alternates: {
    canonical: "https://databasus.com/sponsorship",
  },
  openGraph: {
    type: "website",
    url: "https://databasus.com/sponsorship",
    title: "Sponsor Databasus | Support open-source PostgreSQL backups",
    description:
      "Databasus is free forever under Apache 2.0. Sponsorship funds the maintenance, security work and new features that everyone gets for free.",
    images: [
      {
        url: "https://databasus.com/images/index/rostislav.png",
        alt: "Rostislav Dugin, developer of Databasus",
        width: 160,
        height: 160,
      },
    ],
    siteName: "Databasus",
    locale: "en_US",
  },
  twitter: {
    card: "summary_large_image",
    title: "Sponsor Databasus | Support open-source PostgreSQL backups",
    description:
      "Databasus is free forever under Apache 2.0. Sponsorship funds the maintenance, security work and new features that everyone gets for free.",
    images: ["https://databasus.com/images/index/rostislav.png"],
  },
  applicationName: "Databasus",
  appleWebApp: {
    title: "Databasus",
    capable: true,
  },
  icons: {
    icon: [
      { url: "/favicon.ico", type: "image/x-icon" },
      { url: "/favicon.svg", type: "image/svg+xml" },
    ],
    apple: "/favicon.svg",
    shortcut: "/favicon.ico",
  },
};

export default function SponsorshipPage() {
  return (
    <div className="overflow-x-hidden">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "WebPage",
            name: "Sponsor Databasus",
            description:
              "Databasus is free forever under Apache 2.0. Sponsorship funds the maintenance, security work and new features that everyone gets for free.",
            url: "https://databasus.com/sponsorship",
          }),
        }}
      />

      {PADDLE_ENABLED && <PaddleInitComponent />}

      {/* HEADER */}
      <header className="fixed top-0 left-0 right-0 z-50 flex justify-center pt-3 md:pt-5 px-4 md:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <nav className="relative flex items-center justify-between border backdrop-blur-md bg-[#0C0E13]/80 md:bg-[#0C0E13]/20 border-[#ffffff20] px-3 py-2 rounded-xl">
            <a href="/" className="flex items-center gap-2.5">
              <img
                src="/logo.svg"
                alt="Databasus logo"
                width={32}
                height={32}
                className="h-7 w-7 md:h-8 md:w-8"
                fetchPriority="high"
                loading="eager"
              />
              <span className="text-base md:text-lg font-semibold pl-1">
                Databasus
              </span>
            </a>

            {/* Desktop Navigation */}
            <div className="absolute left-1/2 -translate-x-1/2 hidden lg:flex items-center gap-3">
              <a
                href="/#how-to-use"
                className="py-2 hover:text-gray-300 transition-colors"
              >
                How to use
              </a>

              <a
                href="/installation"
                className="py-2 hover:text-gray-300 transition-colors"
              >
                Docs
              </a>

              <a
                href="https://t.me/databasus_community"
                target="_blank"
                rel="noopener noreferrer"
                className="py-2 hover:text-gray-300 transition-colors"
              >
                Community
              </a>

              <a
                href="/sponsorship"
                className="py-2 hover:text-gray-300 transition-colors"
              >
                Sponsorship
              </a>
            </div>

            {/* GitHub Button */}
            <a
              href="https://github.com/databasus/databasus"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-2 hover:opacity-70 rounded-lg px-2 md:px-3 py-2 text-[14px] border border-[#ffffff20] bg-[#0C0E13] transition-colors"
            >
              <svg
                aria-hidden={true}
                width="24"
                height="24"
                viewBox="0 0 20 20"
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
              >
                <g clipPath="url(#clip0_sponsor_nav)">
                  <path
                    fillRule="evenodd"
                    clipRule="evenodd"
                    d="M9.9702 0C4.45694 0 0 4.4898 0 10.0443C0 14.4843 2.85571 18.2427 6.81735 19.5729C7.31265 19.6729 7.49408 19.3567 7.49408 19.0908C7.49408 18.858 7.47775 18.0598 7.47775 17.2282C4.70429 17.8269 4.12673 16.0308 4.12673 16.0308C3.68102 14.8667 3.02061 14.5676 3.02061 14.5676C2.11286 13.9522 3.08673 13.9522 3.08673 13.9522C4.09367 14.0188 4.62204 14.9833 4.62204 14.9833C5.51327 16.5131 6.94939 16.0808 7.52714 15.8147C7.60959 15.1661 7.87388 14.7171 8.15449 14.4678C5.94245 14.2349 3.6151 13.3702 3.6151 9.51204C3.6151 8.41449 4.01102 7.51653 4.63837 6.81816C4.53939 6.56878 4.19265 5.53755 4.73755 4.15735C4.73755 4.15735 5.57939 3.89122 7.47755 5.18837C8.29022 4.9685 9.12832 4.85666 9.9702 4.85571C10.812 4.85571 11.6702 4.97225 12.4627 5.18837C14.361 3.89122 15.2029 4.15735 15.2029 4.15735C15.7478 5.53755 15.4008 6.56878 15.3018 6.81816C15.9457 7.51653 16.3253 8.41449 16.3253 9.51204C16.3253 13.3702 13.998 14.2182 11.7694 14.4678C12.1327 14.7837 12.4461 15.3822 12.4461 16.3302C12.4461 17.6771 12.4298 18.7582 12.4298 19.0906C12.4298 19.3567 12.6114 19.6729 13.1065 19.5731C17.0682 18.2424 19.9239 14.4843 19.9239 10.0443C19.9402 4.4898 15.4669 0 9.9702 0Z"
                    fill="white"
                  />
                </g>
                <defs>
                  <clipPath id="clip0_sponsor_nav">
                    <rect width="20" height="20" fill="white" />
                  </clipPath>
                </defs>
              </svg>
              <span className="hidden xl:inline">
                Star on GitHub, it&apos;s really important ❤️
              </span>
              <span className="inline xl:hidden">GitHub</span>
            </a>
          </nav>
        </div>
      </header>

      {/* MAIN */}
      <main className="relative overflow-hidden pt-[60px] md:pt-[68px]">
        <div className="relative mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px] px-4 md:px-6 lg:px-0 pt-12 md:pt-[80px] pb-12 md:pb-[100px]">
          {/* Background ellipse */}
          <div className="relative">
            <div className="absolute left-1/2 -translate-x-1/2 -translate-y-1/4 w-[400px] h-[400px] md:w-[900px] md:h-[900px] bg-[#155dfc]/4 top-0 rounded-full blur-3xl -z-10" />
          </div>

          {/* ===== HERO — two columns: pitch (left) + pricing (right) ===== */}
          <section className="grid grid-cols-1 lg:grid-cols-[1.05fr_0.95fr] gap-10 lg:gap-14 items-start">
            {/* LEFT — pitch */}
            <div className="text-left">
              {/* Identity — photo with name beside it */}
              <div className="flex items-center gap-4">
                <div className="relative shrink-0">
                  <div
                    aria-hidden="true"
                    className="pointer-events-none absolute -inset-1.5 rounded-full bg-[#155dfc]/25 blur-2xl"
                  />
                  <img
                    src="/images/index/rostislav.png"
                    width={72}
                    height={72}
                    loading="eager"
                    className="relative h-16 w-16 md:h-[72px] md:w-[72px] rounded-full object-cover ring-1 ring-[#ffffff20]"
                    alt="Rostislav Dugin"
                  />
                </div>
                <div className="min-w-0">
                  <div className="text-xl md:text-2xl font-bold leading-tight text-white">
                    Rostislav Dugin
                  </div>
                  <div className="mt-1 flex items-center gap-1.5 text-sm text-gray-400">
                    Developer of Databasus
                  </div>
                </div>
              </div>

              <h1 className="mt-8 text-3xl max-w-[450px] md:text-4xl font-extrabold leading-[1.1] tracking-tight text-white">
                Databasus helps you? You can help Databasus as well
              </h1>
              <p className="mt-4 text-lg md:text-xl leading-relaxed text-gray-300">
                Sponsor the work behind it — and keep it free, maintained and
                independent. This is your investment in open source
              </p>

              {/* The pledge */}
              <figure className="mt-8 rounded-2xl border border-[#155dfc]/30 bg-[#155dfc]/[0.06] p-6 md:p-7">
                <figcaption className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-[#155dfc]">
                  <span className="h-1.5 w-1.5 rounded-full bg-[#155dfc]" />
                  The promise
                </figcaption>
                <blockquote className="mt-3">
                  <p className="text-lg md:text-xl font-extrabold leading-snug text-white">
                    Databasus is open — and it always will be.
                  </p>
                  <p className="mt-3 text-base md:text-lg leading-relaxed text-gray-300">
                    No open core. No feature gates. No paywalled features. Every
                    capability, free for everyone, forever, under Apache 2.0
                  </p>
                </blockquote>
              </figure>
            </div>

            {/* RIGHT — pricing panel (sticky on desktop) */}
            <div className="lg:sticky lg:top-28">
              <div
                id="pricing-panel"
                className="rounded-2xl border border-[#ffffff20] bg-[#0C0E13] p-5 md:p-6"
              >
                <h2 className="text-lg md:text-xl font-bold text-white">
                  What’s your saved data worth?
                </h2>
                <p className="mt-1 text-sm text-gray-400">
                  Every sponsor genuinely keeps Databasus going — and they are
                  featured on the homepage and in our GitHub repo.
                </p>

                {/* Monthly / Annual billing toggle */}
                <div className="mt-4 grid grid-cols-2 gap-1 rounded-xl border border-[#ffffff20] p-1">
                  <button
                    type="button"
                    data-billing="monthly"
                    aria-label="Show monthly pricing"
                    className="cursor-pointer rounded-lg bg-[#155dfc] px-3 py-2.5 text-sm font-semibold text-white transition-colors [.is-annual_&]:bg-transparent [.is-annual_&]:text-gray-400"
                  >
                    Monthly
                  </button>
                  <button
                    type="button"
                    data-billing="annual"
                    aria-label="Show annual pricing, two months free"
                    className="flex cursor-pointer items-center justify-center gap-1.5 rounded-lg px-3 py-2.5 text-sm font-semibold text-gray-400 transition-colors [.is-annual_&]:bg-[#155dfc] [.is-annual_&]:text-white"
                  >
                    Annual
                    <span className="rounded-full bg-[#155dfc]/15 px-1.5 py-0.5 text-[10px] font-semibold uppercase leading-none tracking-wide text-[#155dfc] [.is-annual_&]:bg-white/20 [.is-annual_&]:text-white">
                      −2 mo
                    </span>
                  </button>
                </div>

                {/* Sponsor display name — passed to Paddle as custom data so we
                    list the right name (person or company) in our sponsors. */}
                <input
                  id="sponsor-name"
                  type="text"
                  placeholder="Name to list you as a sponsor (you or your company)"
                  className="mt-4 w-full rounded-xl border border-[#ffffff20] bg-[#0C0E13] px-3 py-2.5 text-sm text-white placeholder:text-gray-500 focus:border-[#155dfc] focus:outline-none"
                />

                <div className="mt-3 grid grid-cols-2 gap-2.5">
                  {TIERS.map((tier, i) => {
                    const spanFull =
                      TIERS.length % 2 === 1 && i === TIERS.length - 1;
                    const cardClass = `group relative flex flex-col justify-center rounded-xl border px-3.5 py-3 transition-all hover:-translate-y-0.5 hover:border-[#155dfc] hover:bg-[#155dfc]/[0.08] ${
                      tier.popular
                        ? "border-[#155dfc] bg-[#155dfc]/[0.08]"
                        : "border-[#ffffff20] bg-[#0C0E13]"
                    } ${spanFull ? "col-span-2 items-center text-center" : ""} ${
                      tier.annual ? "" : "[.is-annual_&]:hidden"
                    }`;

                    // Full-card clickable overlay for one billing period.
                    const overlay = (
                      price: { price: string; priceId: string },
                      visibility: string,
                    ) =>
                      priceUsesPaddle(price.priceId) ? (
                        <a
                          className={`js-sponsor-checkout absolute inset-0 z-10 cursor-pointer rounded-xl ${visibility}`}
                          data-items={JSON.stringify([
                            { priceId: price.priceId, quantity: 1 },
                          ])}
                          aria-label={`Sponsor at the ${tier.name} tier`}
                        />
                      ) : (
                        <a
                          href={TELEGRAM_URL}
                          target="_blank"
                          rel="noopener noreferrer"
                          className={`absolute inset-0 z-10 cursor-pointer rounded-xl ${visibility}`}
                          aria-label={`Sponsor at the ${tier.name} tier`}
                        />
                      );

                    return (
                      <div key={tier.name} className={cardClass}>
                        {tier.popular && (
                          <span className="absolute -top-2 right-2.5 z-20 rounded-full bg-[#155dfc] px-2 py-0.5 text-[10px] font-semibold uppercase leading-none tracking-wide text-white">
                            Popular
                          </span>
                        )}
                        <span className="text-xs font-medium text-gray-400">
                          {tier.name}
                        </span>

                        {/* Monthly price */}
                        <span className="mt-0.5 flex items-baseline gap-0.5 [.is-annual_&]:hidden">
                          <span className="text-xl font-extrabold tabular-nums text-white">
                            {tier.monthly.price}
                          </span>
                          <span className="text-xs text-gray-500">/mo</span>
                        </span>

                        {/* Annual price */}
                        {tier.annual && (
                          <span className="mt-0.5 hidden items-baseline gap-0.5 [.is-annual_&]:flex">
                            <span className="text-xl font-extrabold tabular-nums text-white">
                              {tier.annual.price}
                            </span>
                            <span className="text-xs text-gray-500">/yr</span>
                          </span>
                        )}

                        {overlay(tier.monthly, "[.is-annual_&]:hidden")}
                        {tier.annual &&
                          overlay(tier.annual, "hidden [.is-annual_&]:block")}
                      </div>
                    );
                  })}
                </div>

                <p className="mt-5 text-sm text-gray-400">
                  Need a custom amount or invoice terms?{" "}
                  <a
                    href="mailto:info@databasus.com"
                    className="font-medium text-[#155dfc] hover:underline"
                  >
                    info@databasus.com
                  </a>
                </p>

                {/* Toggle behaviour — vanilla JS, no client component needed */}
                <Script id="billing-toggle" strategy="afterInteractive">{`
                  (function(){
                    var p=document.getElementById('pricing-panel');
                    if(!p)return;
                    document.querySelectorAll('[data-billing]').forEach(function(b){
                      b.addEventListener('click',function(){
                        p.classList.toggle('is-annual', b.getAttribute('data-billing')==='annual');
                      });
                    });
                  })();
                `}</Script>

                {/* Open the Paddle overlay ourselves so we can attach the
                    sponsor name typed above as customData on the transaction. */}
                <Script id="sponsor-checkout" strategy="afterInteractive">{`
                  (function(){
                    document.querySelectorAll('.js-sponsor-checkout').forEach(function(el){
                      el.addEventListener('click',function(e){
                        e.preventDefault();
                        var P=window.Paddle;
                        if(!P||!P.Checkout)return;
                        var input=document.getElementById('sponsor-name');
                        var name=input?input.value.trim():'';
                        var open={items:JSON.parse(el.getAttribute('data-items'))};
                        if(name)open.customData={sponsor_name:name};
                        P.Checkout.open(open);
                      });
                    });
                  })();
                `}</Script>
              </div>
            </div>
          </section>

          {/* ===== WHY / WHERE IT GOES ===== */}
          <section className="mx-auto w-full max-w-[820px] mt-20 md:mt-28">
            <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold text-white">
              Why your support genuinely matters
            </h2>
            <p className="mt-4 text-base md:text-lg leading-relaxed text-gray-300">
              Keeping Databasus maintained, secure and up to date is real,
              continuous work. This isn’t a tip jar —{" "}
              <strong className="font-semibold text-white">
                every sponsor directly funds the work that keeps the project
                alive and independent
              </strong>{" "}
              with nothing held back for a paid edition, because there isn’t
              one.
            </p>

            <div className="mt-8 grid grid-cols-1 sm:grid-cols-2 border border-[#ffffff20] rounded-xl overflow-hidden">
              {[
                "Maintenance, bug fixes and new PostgreSQL version support",
                "Security work and timely patches",
                "New features — shipped to everyone, free",
                "Independence, with no pressure to gate or relicense",
              ].map((item, i) => (
                <div
                  key={item}
                  className={`flex items-start gap-3 p-5 md:p-6 border-[#ffffff20] ${
                    i % 2 === 0 ? "sm:border-r" : ""
                  } ${i < 2 ? "border-b" : ""}`}
                >
                  <svg
                    className="mt-0.5 h-5 w-5 shrink-0 text-[#155dfc]"
                    viewBox="0 0 20 20"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth={2}
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    aria-hidden="true"
                  >
                    <path d="M4 10.5 8 14.5 16 5.5" />
                  </svg>
                  <span className="text-base md:text-lg text-gray-300">
                    {item}
                  </span>
                </div>
              ))}
            </div>
          </section>

          {/* ===== FAQ ===== */}
          <section className="mx-auto w-full max-w-[820px] mt-20 md:mt-28">
            <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold text-white">
              A few questions people ask
            </h2>
            <div className="mt-6 grid grid-cols-1 md:grid-cols-2 gap-4">
              {[
                {
                  q: "Do sponsors get private or exclusive features?",
                  a: "No. Every feature is free for everyone, always. Sponsorship funds the open project — it doesn’t unlock anything, because nothing is locked.",
                },
                {
                  q: "Do sponsors get any recognition?",
                  a: "Yes. Sponsors are listed in the project’s GitHub README and on the Databasus homepage, with a link back to your site or profile. Prefer to stay anonymous? Just let me know.",
                },
                {
                  q: "How do I cancel?",
                  a: "Anytime, self-serve, through the billing portal linked in your receipt email. Your sponsorship stays active until the end of the period you’ve paid for.",
                },
              ].map((item) => (
                <div
                  key={item.q}
                  className="rounded-xl border border-[#ffffff20] p-5 md:p-6"
                >
                  <h3 className="text-base md:text-lg font-bold text-white">
                    {item.q}
                  </h3>
                  <p className="mt-2 text-sm md:text-base text-gray-400">
                    {item.a}
                  </p>
                </div>
              ))}
            </div>
          </section>

          {/* ===== THANK YOU / SIGN-OFF ===== */}
          <section className="mx-auto w-full max-w-[820px] mt-20 md:mt-28">
            <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold text-white">
              Thank you
            </h2>
            <p className="mt-4 text-base md:text-lg leading-relaxed text-gray-300">
              I’d love to keep Databasus free, well-maintained and independent
              for years. Your sponsorship is what makes that possible.
            </p>
            <p className="mt-4 text-base md:text-lg text-gray-300">
              —{" "}
              <strong className="font-semibold text-white">
                Rostislav Dugin
              </strong>
              , developer of Databasus
            </p>
            <div className="mt-4 flex flex-wrap items-center gap-x-6 gap-y-2 text-base">
              <a
                href={TELEGRAM_URL}
                target="_blank"
                rel="noopener noreferrer"
                className="font-medium text-[#155dfc] hover:underline"
              >
                Telegram
              </a>
              <a
                href="mailto:info@databasus.com"
                className="font-medium text-[#155dfc] hover:underline"
              >
                Email
              </a>
              <a
                href="https://rostislav-dugin.com"
                target="_blank"
                rel="noopener noreferrer"
                className="font-medium text-[#155dfc] hover:underline"
              >
                CV
              </a>
            </div>
          </section>
        </div>
      </main>

      {/* FOOTER */}
      <footer className="py-8 md:py-12 border-t border-[#ffffff20] px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col items-center">
            <a href="/" className="flex items-center gap-2.5 mb-6">
              <img
                src="/logo.svg"
                alt="Databasus logo"
                width={32}
                height={32}
                className="h-7 w-7 md:h-8 md:w-8"
              />
              <span className="text-base md:text-lg font-semibold">
                Databasus
              </span>
            </a>

            <div className="flex flex-col gap-3 mb-4 text-sm md:text-base">
              {/* First row - Database backup links */}
              <div className="flex flex-wrap items-center justify-center gap-4 md:gap-6">
                <a href="/" className="hover:text-gray-200 transition-colors">
                  PostgreSQL backup
                </a>
                <a
                  href="/mysql-backup"
                  className="hover:text-gray-200 transition-colors"
                >
                  MySQL and MariaDB backup
                </a>
                <a
                  href="/mongodb-backup"
                  className="hover:text-gray-200 transition-colors"
                >
                  MongoDB backup
                </a>
              </div>

              {/* Second row - General links */}
              <div className="flex flex-wrap items-center justify-center gap-4 md:gap-6">
                <a
                  href="/installation"
                  className="hover:text-gray-200 transition-colors"
                >
                  Documentation
                </a>
                <a
                  href="https://github.com/databasus/databasus"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="hover:text-gray-200 transition-colors"
                >
                  GitHub
                </a>
                <a
                  href="https://t.me/databasus_community"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="hover:text-gray-200 transition-colors"
                >
                  Community
                </a>
                <a
                  href="/sponsorship"
                  className="hover:text-gray-200 transition-colors"
                >
                  Sponsorship
                </a>
                <a
                  href="https://rostislav-dugin.com"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="hover:text-gray-200 transition-colors"
                >
                  Developer
                </a>
              </div>

              {/* Third row - Legal links */}
              <div className="flex flex-wrap items-center justify-center gap-4 md:gap-6">
                <a
                  href="/privacy"
                  className="hover:text-gray-200 transition-colors"
                >
                  Privacy policy
                </a>
                <a
                  href="/terms-of-use"
                  className="hover:text-gray-200 transition-colors"
                >
                  Terms of use
                </a>
              </div>
            </div>

            <a
              href="mailto:info@databasus.com"
              className="hover:text-gray-200 transition-colors text-sm md:text-base mb-4"
            >
              info@databasus.com
            </a>

            <p className="text-gray-400 text-sm md:text-base text-center">
              © 2026 Databasus™. All rights reserved.
            </p>
          </div>
        </div>
      </footer>
    </div>
  );
}
