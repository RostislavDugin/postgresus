import type { Metadata } from "next";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Terms of Use - Databasus",
  description:
    "Terms of use for Databasus paid services, sold through Paddle as merchant of record. Read about billing, refunds and cancellation.",
  alternates: {
    canonical: "https://databasus.com/terms-of-use",
  },
  robots: "index, follow",
};

export default function TermsOfUsePage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "WebPage",
            headline: "Terms of Use - Databasus",
            description:
              "Terms of use for Databasus paid services, sold through Paddle as merchant of record.",
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
        <DocsSidebarComponent />

        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="terms-of-use">Terms of Use</h1>

              <p className="text-lg text-gray-400">Last updated: October 23, 2026</p>

              <p>
                These Terms of Use (&quot;Terms&quot;) govern purchases of paid
                services from Databasus through the Databasus website at{" "}
                <a
                  href="https://databasus.com"
                  className="text-blue-500 hover:text-blue-600"
                >
                  databasus.com
                </a>
                . Databasus is operated by
                Databasus (IE Rostyslav Duhin, Identification Number: 347010209),
                registered in Georgia (&quot;we&quot;, &quot;us&quot;,
                &quot;our&quot;).
              </p>

              <p>
                For details on how we handle your personal data, see our{" "}
                <a
                  href="/privacy"
                  className="text-blue-500 hover:text-blue-600"
                >
                  privacy policy
                </a>
                .
              </p>

              <h2 id="acceptance">1. Acceptance of terms</h2>

              <p>
                By purchasing or using a paid service, you agree to be bound by
                these Terms. If you do not agree, do not purchase. If you are
                purchasing on behalf of a company or other organization, you
                represent that you have the authority to bind that organization
                to these Terms.
              </p>

              <p>
                Payments are sold and processed by{" "}
                <strong>Paddle.com Market Limited (&quot;Paddle&quot;)</strong>,
                acting as our authorized reseller and{" "}
                <strong>merchant of record</strong>. When you purchase, your
                purchase is also subject to Paddle&apos;s{" "}
                <a
                  href="https://www.paddle.com/legal/checkout-buyer-terms"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-500 hover:text-blue-600"
                >
                  buyer terms
                </a>
                . Paddle handles the transaction, applicable taxes (VAT/sales
                tax) and issues your invoice.
              </p>

              <h2 id="open-source-software">2. The open source software</h2>

              <p>
                The self-hosted Databasus software is open source and governed
                solely by its{" "}
                <strong>Apache 2.0 license</strong>, not by these Terms. These
                Terms apply only to paid services you purchase from us.
              </p>

              <h2 id="payment">3. Payment and billing</h2>

              <ul>
                <li>
                  Paid services are billed on a <strong>monthly</strong> cycle
                  and renew automatically until cancelled
                </li>
                <li>
                  The price is shown to you before you complete a purchase
                </li>
                <li>
                  Paddle, as merchant of record, is responsible for charging your
                  payment method, calculating and collecting any applicable taxes
                  and issuing your invoice or receipt
                </li>
                <li>
                  Your card and payment details are handled solely by Paddle. We
                  never receive or store your full payment details
                </li>
                <li>
                  If a renewal payment fails, the paid service may be paused or
                  stopped until payment is resolved
                </li>
              </ul>

              <h2 id="refund-policy">4. Refunds and cancellation</h2>

              <ul>
                <li>
                  You may cancel and request a <strong>full refund</strong>{" "}
                  within <strong>14 days</strong> of your{" "}
                  <strong>initial</strong> purchase, without giving any reason,
                  in line with the EU consumer right of withdrawal
                </li>
                <li>
                  You can cancel at any time. Cancellation is self-serve through
                  the billing portal linked in your Paddle receipt email. The
                  paid service remains active until the end of the period you
                  have already paid for
                </li>
                <li>
                  After the 14-day window, there are{" "}
                  <strong>no refunds</strong> for the unused portion of a billing
                  period
                </li>
                <li>
                  The 14-day refund right applies to the initial purchase only.
                  It does <strong>not</strong> reset on each monthly automatic
                  renewal
                </li>
                <li>
                  To request a refund, contact us at{" "}
                  <a
                    href="mailto:info@databasus.com"
                    className="text-blue-500 hover:text-blue-600"
                  >
                    info@databasus.com
                  </a>{" "}
                  or Paddle via your receipt. Refunds are returned to the
                  original payment method
                </li>
              </ul>

              <h2 id="disclaimer-of-warranties">5. Disclaimer of warranties</h2>

              <p>
                <strong>
                  Paid services and the website are provided &quot;AS IS&quot;
                  and &quot;AS AVAILABLE&quot; without warranties of any kind,
                  whether express, implied or statutory.
                </strong>{" "}
                We provide them on a best-effort basis and do not warrant that
                they will be uninterrupted, error-free, continuously available
                or free of harmful components.
              </p>

              <h2 id="limitation-of-liability">6. Limitation of liability</h2>

              <p>
                <strong>
                  To the maximum extent permitted by applicable law, we shall not
                  be liable for any indirect, incidental, special, consequential
                  or punitive damages, including but not limited to loss of
                  profits, business, revenue, goodwill or anticipated benefit,
                  arising out of or related to your purchase or use of a paid
                  service, regardless of the cause or theory of liability.
                </strong>
              </p>

              <p>
                Our total aggregate liability for any and all claims arising out
                of or related to a paid service shall not exceed the total fees
                you paid in the three (3) months immediately preceding the event
                giving rise to the claim. This limitation applies to the fullest
                extent permitted by law, even if we have been advised of the
                possibility of such damages.
              </p>

              <h2 id="term-and-termination">7. Term and termination</h2>

              <p>
                Your subscription to a paid service continues until you cancel
                or it is terminated. You may cancel at any time as described in
                the refund and cancellation section above.
              </p>

              <p>
                We may end a subscription at our discretion, including for
                breach of these Terms, non-payment, suspected fraud or abuse, or
                if continuing to offer the paid service is no longer practical.
                Where we end a subscription for our own reasons (not for a
                breach by you), we will refund the prepaid, unused portion of
                the current billing period.
              </p>

              <p>
                Sections that by their nature should survive termination
                (including disclaimer of warranties, limitation of liability and
                governing law) shall survive.
              </p>

              <h2 id="modifications">8. Changes to terms and pricing</h2>

              <p>
                We may modify these Terms from time to time. The &quot;Last
                updated&quot; date at the top indicates when the Terms were last
                revised. Your continued use of a paid service after changes take
                effect constitutes acceptance of the modified Terms. If you do
                not agree with the changes, you may cancel before they take
                effect.
              </p>

              <p>
                We may change pricing at any time. Pricing changes take effect
                at the start of your next billing cycle. You may cancel before
                the next billing cycle if you do not agree with the new pricing.
              </p>

              <h2 id="governing-law">9. Governing law and disputes</h2>

              <p>
                These Terms shall be governed by and construed in accordance with
                the laws of Georgia. Any disputes arising out of or relating to
                these Terms shall be submitted to the exclusive jurisdiction of
                the courts of Georgia.
              </p>

              <p>
                If you are a consumer in the European Union, nothing in these
                Terms affects your rights under the mandatory consumer protection
                laws of your country of residence.
              </p>

              <p>
                Before initiating any legal proceedings, you agree to first
                contact us at{" "}
                <a
                  href="mailto:info@databasus.com"
                  className="text-blue-500 hover:text-blue-600"
                >
                  info@databasus.com
                </a>{" "}
                and attempt to resolve the dispute informally for at least 30
                days.
              </p>

              <h2 id="contact">Contact</h2>

              <p>If you have questions about these Terms, contact us:</p>

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
                    databasus.com
                  </a>
                </li>
                <li>
                  <strong>Operator:</strong> Databasus (IE Rostyslav Duhin),
                  Georgia
                </li>
              </ul>
            </article>
          </div>
        </main>

        <DocTableOfContentComponent />
      </div>
    </>
  );
}
