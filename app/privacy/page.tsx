import type { Metadata } from "next";
import DocsNavbarComponent from "../components/DocsNavbarComponent";
import DocsSidebarComponent from "../components/DocsSidebarComponent";
import DocTableOfContentComponent from "../components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Privacy Policy - Databasus",
  description:
    "Learn how Databasus respects your privacy with anonymous analytics, GDPR compliance and no personal data collection. We use privacy-first analytics to improve user experience.",
  keywords: [
    "privacy policy",
    "GDPR compliance",
    "anonymous analytics",
    "privacy-compliant",
    "Rybbit.io",
    "no tracking",
    "data privacy",
    "user privacy",
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
                Last updated: December 26, 2025
              </p>

              <p>
                At Databasus, we take your privacy seriously. This privacy
                policy explains how we handle data when you visit our website.
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
