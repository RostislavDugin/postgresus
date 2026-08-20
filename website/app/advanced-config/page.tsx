import type { Metadata } from "next";
import DocsNavbarComponent from "../components/DocsNavbarComponent";
import DocsSidebarComponent from "../components/DocsSidebarComponent";
import DocTableOfContentComponent from "../components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Advanced config - Databasus Documentation",
  description:
    "Optional environment variables for self-hosting Databasus: Google and GitHub sign-in, SMTP email, Cloudflare Turnstile captcha, telemetry, log shipping and a custom analytics script. Not needed for a default install.",
  keywords: [
    "Databasus environment variables",
    "Databasus advanced configuration",
    "self-hosted configuration",
    "GitHub OAuth",
    "Google OAuth",
    "SMTP email setup",
    "Cloudflare Turnstile",
    "Docker environment variables",
  ],
  openGraph: {
    title: "Advanced config - Databasus Documentation",
    description:
      "Optional environment variables for self-hosting Databasus: Google and GitHub sign-in, SMTP email, Cloudflare Turnstile captcha, telemetry, log shipping and a custom analytics script. Not needed for a default install.",
    type: "article",
    url: "https://databasus.com/advanced-config",
  },
  twitter: {
    card: "summary",
    title: "Advanced config - Databasus Documentation",
    description:
      "Optional environment variables for self-hosting Databasus: Google and GitHub sign-in, SMTP email, Cloudflare Turnstile captcha, telemetry, log shipping and a custom analytics script. Not needed for a default install.",
  },
  alternates: {
    canonical: "https://databasus.com/advanced-config",
  },
  robots: "index, follow",
};

export default function AdvancedConfigPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Advanced config - Databasus Documentation",
            description:
              "Optional environment variables for self-hosting Databasus: Google and GitHub sign-in, SMTP email, Cloudflare Turnstile captcha, telemetry, log shipping and a custom analytics script. Not needed for a default install.",
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
              <h1 id="advanced-config">Advanced config</h1>

              <p className="text-lg text-gray-400">
                Databasus runs with sensible defaults out of the box — a
                standard single-container install needs no configuration at all.
                Every variable on this page is <strong>optional</strong> and not
                needed in 99% of production setups
              </p>

              <h2 id="oauth">OAuth</h2>

              <p>
                By default Databasus uses email and password sign-in. You can
                additionally let people sign in with their Google or GitHub
                account. A provider&apos;s button appears as soon as its client
                ID is set, but sign-in only completes when <strong>both</strong>{" "}
                the client ID and the client secret are present.
              </p>

              <p>
                When you register the OAuth application, set its redirect
                (callback) URL to{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>. Because
                of that redirect, OAuth sign-in needs your instance served over
                HTTPS on a public domain — see the note below.
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-blue-500 mt-0.5 shrink-0"
                >
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 16v-4M12 8h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <strong>HTTPS is required for sign-in and email.</strong>{" "}
                    OAuth sign-in and email both need your instance reachable
                    over HTTPS on a public domain — OAuth providers redirect the
                    browser back to{" "}
                    <code>https://&lt;your-domain&gt;/auth/callback</code>, and
                    links inside emails must open for whoever receives them. A
                    localhost-only or plain-HTTP instance cannot use these
                    features. The simplest way to get HTTPS is the{" "}
                    <a
                      href="/installation/#caddy-reverse-proxy"
                      className="text-blue-400 hover:text-blue-300"
                    >
                      Caddy reverse proxy
                    </a>{" "}
                    setup.
                  </p>
                </div>
              </div>

              <h3 id="oauth-google">Google</h3>

              <p>
                Create an OAuth client in the{" "}
                <a
                  href="https://console.cloud.google.com/apis/credentials"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Google Cloud Console
                </a>{" "}
                (APIs &amp; Services → Credentials → Create credentials → OAuth
                client ID, application type <em>Web application</em>) and add{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code> as an
                authorized redirect URI.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_ID</code>
                    </td>
                    <td data-label="Description">
                      Client ID of your Google OAuth client. Setting it shows
                      the &quot;Sign in with Google&quot; button.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Description">
                      Client secret of your Google OAuth client. Required
                      together with the ID for sign-in to work.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h3 id="oauth-github">GitHub</h3>

              <p>
                Create an OAuth app under{" "}
                <a
                  href="https://github.com/settings/developers"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  GitHub Developer settings
                </a>{" "}
                (Settings → Developer settings → OAuth Apps → New OAuth App) and
                set the authorization callback URL to{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_ID</code>
                    </td>
                    <td data-label="Description">
                      Client ID of your GitHub OAuth app. Setting it shows the
                      &quot;Sign in with GitHub&quot; button.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Description">
                      Client secret of your GitHub OAuth app. Required together
                      with the ID for sign-in to work.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="email-smtp">Email (SMTP)</h2>

              <p>
                Connect an SMTP server so Databasus can send transactional email
                such as password-reset links and workspace invitations. Email is
                treated as configured{" "}
                <strong>
                  only when both <code>SMTP_HOST</code> and{" "}
                  <code>DATABASUS_URL</code> are set
                </strong>{" "}
                — until then, email features stay hidden in the UI.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>SMTP_HOST</code>
                    </td>
                    <td data-label="Description">
                      SMTP server hostname (e.g. <code>smtp.gmail.com</code>).
                      Enables email together with <code>DATABASUS_URL</code>.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PORT</code>
                    </td>
                    <td data-label="Description">
                      SMTP server port (e.g. <code>587</code>). Must be a
                      positive integer when <code>SMTP_HOST</code> is set.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_USER</code>
                    </td>
                    <td data-label="Description">
                      Username for SMTP authentication.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PASSWORD</code>
                    </td>
                    <td data-label="Description">
                      Password for SMTP authentication. For Gmail, use an App
                      Password — not your account password.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_FROM</code>
                    </td>
                    <td data-label="Description">
                      The &quot;From&quot; address on outgoing email.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_INSECURE_SKIP_VERIFY</code>
                    </td>
                    <td data-label="Description">
                      Set to <code>true</code> to skip TLS certificate
                      verification when connecting to the SMTP server. Defaults
                      to <code>false</code>. Use it only for servers with a
                      self-signed certificate on a trusted network — it disables
                      protection against man-in-the-middle attacks.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>DATABASUS_URL</code>
                    </td>
                    <td data-label="Description">
                      Public base URL of your instance (e.g.{" "}
                      <code>https://backup.example.com</code>). Used to build
                      links inside emails. Required together with{" "}
                      <code>SMTP_HOST</code>.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="signup-captcha">
                Sign up captcha (Cloudflare Turnstile)
              </h2>

              <p>
                If your instance is reachable from the public internet, you can
                put a{" "}
                <a
                  href="https://www.cloudflare.com/products/turnstile/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Cloudflare Turnstile
                </a>{" "}
                challenge on the sign-up and sign-in forms to keep bots out.
                Both keys come from the Turnstile dashboard, and the challenge
                activates only when both are set.
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-blue-500 mt-0.5 shrink-0"
                >
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 16v-4M12 8h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    To stop external sign-ups entirely rather than just
                    challenging them, you do not need a captcha at all — open{" "}
                    <strong>Databasus settings → Allow sign up</strong> in the
                    UI and turn it off. That closes the sign-up form completely.
                  </p>
                </div>
              </div>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SITE_KEY</code>
                    </td>
                    <td data-label="Description">
                      Public Turnstile site key, used to render the widget in
                      the browser.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SECRET_KEY</code>
                    </td>
                    <td data-label="Description">
                      Secret Turnstile key, used by the backend to validate
                      challenge responses.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="telemetry">Telemetry</h2>

              <p>
                Databasus sends anonymous, non-identifying usage telemetry by
                default. It carries no personal data and helps us understand how
                the project is used. You can read exactly what is collected in
                the{" "}
                <a
                  href="/privacy"
                  className="text-blue-400 hover:text-blue-300"
                >
                  privacy policy
                </a>
                , and you can turn it off completely.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Default</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>IS_DISABLE_ANONYMOUS_TELEMETRY</code>
                    </td>
                    <td data-label="Default">
                      <code>false</code>
                    </td>
                    <td data-label="Description">
                      Set to <code>true</code> to disable anonymous usage
                      telemetry.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="log-shipping">Log shipping</h2>

              <p>
                By default Databasus keeps its application logs inside the
                container. If you run central log aggregation, you can ship them
                to an external VictoriaLogs instance instead. Setting{" "}
                <code>VICTORIA_LOGS_URL</code> enables shipping; the username
                and password are only needed if your endpoint requires basic
                auth.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Default</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>VICTORIA_LOGS_URL</code>
                    </td>
                    <td data-label="Default">—</td>
                    <td data-label="Description">
                      URL of a VictoriaLogs instance to ship application logs
                      to. Leave unset to keep logs in the container.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>VICTORIA_LOGS_USERNAME</code>
                    </td>
                    <td data-label="Default">—</td>
                    <td data-label="Description">
                      Username for the VictoriaLogs endpoint, if it requires
                      basic auth.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>VICTORIA_LOGS_PASSWORD</code>
                    </td>
                    <td data-label="Default">—</td>
                    <td data-label="Description">
                      Password for the VictoriaLogs endpoint, if it requires
                      basic auth.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="analytics-script">Analytics script</h2>

              <p>
                Databasus can inject your own analytics or tracking snippet —
                Google Analytics, Plausible, Umami and similar into the app.
                When <code>ANALYTICS_SCRIPT</code> is set, its value is inserted
                into the page <code>&lt;head&gt;</code> at startup.
              </p>

              <p>
                <strong>Security warning:</strong> the value is injected
                verbatim as raw HTML and JavaScript and runs with full access to
                the Databasus UI in every visitor&apos;s browser. Only ever set
                it to a snippet you fully control and trust.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>ANALYTICS_SCRIPT</code>
                    </td>
                    <td data-label="Description">
                      Custom <code>&lt;script&gt;</code> markup injected before
                      the closing <code>&lt;/head&gt;</code> tag. Leave unset to
                      add no analytics.
                    </td>
                  </tr>
                </tbody>
              </table>

            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
