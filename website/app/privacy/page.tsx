import type { Metadata } from "next";
import DocsNavbarComponent from "../components/DocsNavbarComponent";
import DocsSidebarComponent from "../components/DocsSidebarComponent";
import DocTableOfContentComponent from "../components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Privacy Policy - Databasus",
  description:
    "Learn how Databasus respects your privacy with anonymous analytics, GDPR compliance and no personal data collection. Covers the website, the self-hosted app and sponsorship payments processed by Paddle.",
  keywords: [
    "privacy policy",
    "GDPR compliance",
    "anonymous analytics",
    "privacy-compliant",
    "Rybbit.io",
    "no tracking",
    "data privacy",
    "user privacy",
    "sponsorship",
    "Paddle",
  ],
  openGraph: {
    title: "Privacy Policy - Databasus",
    description:
      "Learn how Databasus respects your privacy with anonymous analytics, GDPR compliance and no personal data collection.",
    type: "article",
    url: "https://databasus.com/privacy",
  },
  twitter: {
    card: "summary",
    title: "Privacy Policy - Databasus",
    description:
      "Learn how Databasus respects your privacy with anonymous analytics, GDPR compliance and no personal data collection.",
  },
  alternates: {
    canonical: "https://databasus.com/privacy",
  },
  robots: "index, follow",
};

export default function PrivacyPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "WebPage",
            headline: "Privacy Policy - Databasus",
            description:
              "Learn how Databasus respects your privacy with anonymous analytics, GDPR compliance and no personal data collection.",
            author: {
              "@type": "Organization",
              name: "Databasus",
            },
            publisher: {
              "@type": "Organization",
              name: "Databasus",
              logo: {
                "@type": "ImageObject",
                url: "https://databasus.com/logo.svg",
              },
            },
          }),
        }}
      />

      <DocsNavbarComponent />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="privacy-policy">Privacy Policy</h1>

              <p className="text-lg text-gray-400">
                Last updated: June 10, 2026
              </p>

              <p>
                At Databasus, we take your privacy seriously. This privacy
                policy explains how we handle data when you visit our website.
              </p>

              <p>
                This policy covers our website, the self-hosted Databasus app
                and sponsorship. If you sponsor Databasus, please also see our{" "}
                <a
                  href="/terms-of-use"
                  className="text-blue-500 hover:text-blue-600"
                >
                  terms of use
                </a>
                .
              </p>

              <h2 id="our-commitment">Our commitment to privacy</h2>

              <p>
                We believe in transparency and respect for user privacy. Our
                approach is simple:{" "}
                <strong>
                  we collect only anonymous, non-personal data to understand how
                  our website is used and to improve the user experience
                </strong>
                .
              </p>

              <h2 id="anonymous-analytics">Anonymous analytics</h2>

              <p>
                To understand how visitors interact with our website and to
                improve the user experience, we use{" "}
                <a
                  href="https://rybbit.io"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-500 hover:text-blue-600"
                >
                  Rybbit.io
                </a>
                , a privacy-compliant analytics service.
              </p>

              <p>
                Rybbit.io is specifically designed to respect user privacy while
                providing valuable insights about website usage. Unlike
                traditional analytics tools, it does not track individual users
                or collect personal information.
              </p>

              <h3 id="why-analytics">Why we use analytics</h3>

              <p>We use analytics to:</p>

              <ul>
                <li>Understand which pages are most valuable to our users</li>
                <li>
                  Identify technical issues and improve website performance
                </li>
                <li>
                  Make data-driven decisions about content and feature
                  development
                </li>
                <li>
                  Measure the effectiveness of our documentation and resources
                </li>
              </ul>

              <h2 id="what-we-collect">What we collect</h2>

              <p>
                We collect only anonymous, aggregated data about website usage:
              </p>

              <ul>
                <li>
                  <strong>Page views</strong> - which pages are visited
                </li>
                <li>
                  <strong>Referral sources</strong> - where visitors came from
                  (e.g., search engines, social media)
                </li>
                <li>
                  <strong>Browser and device type</strong> - general information
                  about the technology used (e.g., Chrome on Windows, Safari on
                  iOS)
                </li>
                <li>
                  <strong>Geographic region</strong> - country or region only,
                  not precise location
                </li>
                <li>
                  <strong>Time on page</strong> - how long visitors spend on
                  each page
                </li>
              </ul>

              <h2 id="what-we-do-not-collect">What we do NOT collect</h2>

              <p>
                We are committed to protecting your privacy. We do NOT collect:
              </p>

              <ul>
                <li>
                  <strong>IP addresses</strong> - your IP address is never
                  logged or stored
                </li>
                <li>
                  <strong>Personal information</strong> - no names, emails,
                  phone numbers or any personally identifiable information
                </li>
                <li>
                  <strong>Precise location data</strong> - we only collect
                  country-level geographic information
                </li>
                <li>
                  <strong>Browser fingerprints</strong> - we do not use
                  fingerprinting techniques to track users
                </li>
                <li>
                  <strong>Cross-site tracking</strong> - we do not track your
                  activity across other websites
                </li>
              </ul>

              <h2 id="user-identification">User identification</h2>

              <p>
                To distinguish between unique visitors without compromising
                privacy, Rybbit.io generates a{" "}
                <strong>randomly generated, salted user ID</strong> for each
                visitor.
              </p>

              <p>This ID:</p>

              <ul>
                <li>Is completely anonymous and cannot be linked to you</li>
                <li>Is randomly generated and cryptographically salted</li>
                <li>
                  Does not contain or derive from any personal information
                </li>
                <li>Cannot be used to identify or track you across websites</li>
                <li>
                  Is used solely for counting unique visitors and session
                  analysis
                </li>
              </ul>

              <h2 id="gdpr-compliance">GDPR & privacy compliance</h2>

              <p>
                Our analytics approach is fully compliant with privacy
                regulations including:
              </p>

              <ul>
                <li>
                  <strong>GDPR</strong> (General Data Protection Regulation) -
                  European Union
                </li>
                <li>
                  <strong>CCPA</strong> (California Consumer Privacy Act) -
                  United States
                </li>
                <li>
                  <strong>PECR</strong> (Privacy and Electronic Communications
                  Regulations) - United Kingdom
                </li>
              </ul>

              <p>
                Rybbit.io is designed to be privacy-compliant by default,
                meaning:
              </p>

              <ul>
                <li>No consent banners are required</li>
                <li>No cookies are used for tracking</li>
                <li>No personal data is processed</li>
                <li>Data cannot be used to identify individuals</li>
              </ul>

              <h2 id="no-cookies">No tracking cookies</h2>

              <p>
                We do <strong>not use cookies</strong> for tracking or analytics
                purposes. Rybbit.io operates without cookies, eliminating the
                need for cookie consent banners and respecting your browser
                settings.
              </p>

              <p>
                This means you can browse our website freely without worrying
                about tracking cookies or having to accept cookie policies.
              </p>

              <h2 id="data-storage">Data storage</h2>

              <p>
                All analytics data is stored on <strong>our own server</strong>.
                We maintain full control over the data and infrastructure,
                ensuring:
              </p>

              <ul>
                <li>Data remains under our direct control</li>
                <li>No third-party access to analytics data</li>
                <li>
                  Data is stored securely with industry-standard practices
                </li>
                <li>
                  Analytics data is retained only as long as necessary for
                  statistical purposes
                </li>
              </ul>

              <p>
                Because all data is anonymous and aggregated, there is no risk
                to individual privacy even in the unlikely event of a data
                breach.
              </p>

              <h2 id="your-rights">Your rights</h2>

              <p>
                Because we do not collect any personal data, there is no
                personal information to access, modify or delete. The anonymous
                analytics data we collect cannot be linked to you as an
                individual.
              </p>

              <p>However, you have the right to:</p>

              <ul>
                <li>
                  <strong>Opt out of analytics</strong> - you can block
                  analytics by using browser extensions or privacy-focused
                  browsers
                </li>
                <li>
                  <strong>Request information</strong> - contact us if you have
                  questions about our data practices
                </li>
                <li>
                  <strong>Report concerns</strong> - notify us if you believe
                  our practices violate privacy regulations
                </li>
              </ul>

              <h2 id="third-party-services">Third-party services</h2>

              <p>
                Our website uses Rybbit.io for analytics. No other third-party
                tracking or analytics services are used. Rybbit.io does not
                share data with any other parties.
              </p>

              <h2 id="anonymous-telemetry">
                Anonymous telemetry in the app
              </h2>

              <p>
                Everything above describes our website. The self-hosted
                Databasus app also collects a small amount of{" "}
                <strong>anonymous telemetry</strong> so we can see how Databasus
                is used in practice. It is intentionally minimal, cannot be tied
                back to you, your data or your instance and can be switched off
                at any time.
              </p>

              <h3 id="telemetry-what-we-collect">What the app collects</h3>

              <p>
                We collect only general, high-level metrics about key features
                — for example which database types and versions are most
                common and which storages and notifiers people enable. It is
                just enough to see the overall picture, nothing more.
              </p>

              <h3 id="telemetry-what-we-do-not-collect">
                What the app does NOT collect
              </h3>

              <p>The telemetry is deliberately coarse. We never collect:</p>

              <ul>
                <li>
                  <strong>Database, table or column names</strong> - the
                  structure and naming of your data stays private
                </li>
                <li>
                  <strong>Names of any kind</strong> - no project, server or
                  account names
                </li>
                <li>
                  <strong>Credentials</strong> - no passwords, tokens or
                  connection strings
                </li>
                <li>
                  <strong>Connection details</strong> - no hostnames, IP
                  addresses or other network information
                </li>
                <li>
                  <strong>Your data</strong> - no backup contents or any actual
                  database data
                </li>
              </ul>

              <p>
                These metrics describe trends across all installations, never a
                single one, so they cannot identify you, your instance or your
                databases.
              </p>

              <h3 id="telemetry-why">Why we collect it</h3>

              <p>
                This helps us decide what to improve first, see which features
                are genuinely useful and spot what is barely used so it can be
                cleaned up. The goal is to keep Databasus as simple and reliable
                as possible.
              </p>

              <h3 id="telemetry-opt-out">How to turn it off</h3>

              <p>
                Anonymous telemetry is fully optional. It can be disabled with a
                single environment variable - see the Telemetry section of the{" "}
                <a
                  href="/advanced-config/#telemetry"
                  className="text-blue-500 hover:text-blue-600"
                >
                  advanced configuration
                </a>{" "}
                page for details.
              </p>

              <h2 id="sponsorship-payments">Sponsorship &amp; payments</h2>

              <p>
                Databasus is free and open source, and sponsorship is an optional
                way to support the project. If you choose to sponsor, here is how
                we handle your data.
              </p>

              <h3 id="sponsorship-paddle">Payments are processed by Paddle</h3>

              <p>
                All sponsorship payments are sold and processed by{" "}
                <strong>Paddle</strong>, acting as our authorized reseller and
                merchant of record. Paddle is the controller of your payment and
                billing data and processes it under its own{" "}
                <a
                  href="https://www.paddle.com/legal/checkout-buyer-terms"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-500 hover:text-blue-600"
                >
                  buyer terms
                </a>
                . Your card and payment details go{" "}
                <strong>only to Paddle</strong> — we never see, receive or store
                your full payment information.
              </p>

              <h3 id="sponsorship-what-we-receive">What we receive</h3>

              <p>
                To fulfil a sponsorship, Paddle shares a limited set of
                information with us, such as your name, email address, company
                name, billing country and the sponsorship tier you chose. We use
                it only to:
              </p>

              <ul>
                <li>Set up and display your sponsor listing</li>
                <li>Provide receipts, invoices and support</li>
                <li>Manage and renew your sponsorship</li>
              </ul>

              <p>
                We do <strong>not</strong> send marketing emails. Any messages we
                send are transactional and directly related to your sponsorship.
              </p>

              <h3 id="sponsorship-public-listing">Your public listing</h3>

              <p>
                A sponsorship includes a listing — which may show your name or
                company name, logo and a link — in the sponsors list. By choosing
                to sponsor and providing these materials, you agree that this
                information is <strong>published publicly</strong> in the sponsors
                list on the website and in the Databasus GitHub repository. You
                can change or remove what is displayed at any time by contacting
                us at{" "}
                <a
                  href="mailto:info@databasus.com"
                  className="text-blue-500 hover:text-blue-600"
                >
                  info@databasus.com
                </a>
                .
              </p>

              <h3 id="sponsorship-retention">Retention and your rights</h3>

              <p>
                We keep your sponsorship-related data for as long as your
                sponsorship is active and afterwards only as required for tax,
                accounting and legal obligations. You keep all the GDPR rights
                described in the{" "}
                <a href="#your-rights" className="text-blue-500 hover:text-blue-600">
                  Your rights
                </a>{" "}
                section above — including access, rectification and erasure — and
                can exercise them by contacting{" "}
                <a
                  href="mailto:info@databasus.com"
                  className="text-blue-500 hover:text-blue-600"
                >
                  info@databasus.com
                </a>
                . Payment records held by Paddle are subject to Paddle&apos;s own
                privacy policy.
              </p>

              <h2 id="changes-to-policy">Changes to this privacy policy</h2>

              <p>
                We may update this privacy policy from time to time to reflect
                changes in our practices or legal requirements. The &quot;Last
                updated&quot; date at the top of this page indicates when the
                policy was last revised.
              </p>

              <p>
                Material changes will be prominently noted on this page. We
                encourage you to review this policy periodically.
              </p>

              <h2 id="contact">Contact information</h2>

              <p>
                If you have any questions, concerns or requests regarding this
                privacy policy or our data practices, please contact us:
              </p>

              <ul>
                <li>
                  <strong>Email:</strong>{" "}
                  <a
                    href="mailto:info@databasus.com"
                    className="text-blue-500 hover:text-blue-600"
                  >
                    info@databasus.com
                  </a>
                </li>
                <li>
                  <strong>Website:</strong>{" "}
                  <a
                    href="https://databasus.com"
                    className="text-blue-500 hover:text-blue-600"
                  >
                    https://databasus.com
                  </a>
                </li>
                <li>
                  <strong>Community:</strong>{" "}
                  <a
                    href="https://t.me/databasus_community"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-500 hover:text-blue-600"
                  >
                    Telegram Community
                  </a>
                </li>
              </ul>

              <h2 id="summary">Summary</h2>

              <p>
                <strong>In short:</strong> We use privacy-compliant anonymous
                analytics to understand how our website is used. We do not
                collect any personal information, IP addresses or use tracking
                cookies. All data is stored on our own server and cannot be used
                to identify individual visitors. Your privacy is fully
                protected.
              </p>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
