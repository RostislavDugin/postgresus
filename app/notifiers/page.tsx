import type { Metadata } from "next";
import Link from "next/link";
import DocsNavbarComponent from "../components/DocsNavbarComponent";
import DocsSidebarComponent from "../components/DocsSidebarComponent";
import DocTableOfContentComponent from "../components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Notifiers - Postgresus Documentation",
  description:
    "List of supported notification channels for Postgresus backup alerts including Slack, Discord, Telegram, Microsoft Teams, Email and Webhooks.",
  keywords: [
    "Postgresus notifiers",
    "backup notifications",
    "Slack notifications",
    "Discord alerts",
    "Telegram notifications",
    "Teams notifications",
    "Email alerts",
    "Webhook notifications",
  ],
  openGraph: {
    title: "Notifiers - Postgresus Documentation",
    description:
      "List of supported notification channels for Postgresus backup alerts including Slack, Discord, Telegram, Microsoft Teams, Email and Webhooks.",
    type: "article",
    url: "https://postgresus.com/notifiers",
  },
  twitter: {
    card: "summary",
    title: "Notifiers - Postgresus Documentation",
    description:
      "List of supported notification channels for Postgresus backup alerts including Slack, Discord, Telegram, Microsoft Teams, Email and Webhooks.",
  },
  alternates: {
    canonical: "https://postgresus.com/notifiers",
  },
  robots: "index, follow",
};

export default function NotifiersPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Notifiers - Postgresus Documentation",
            description:
              "List of supported notification channels for Postgresus backup alerts including Slack, Discord, Telegram, Microsoft Teams, Email and Webhooks.",
            author: {
              "@type": "Organization",
              name: "Postgresus",
            },
            publisher: {
              "@type": "Organization",
              name: "Postgresus",
              logo: {
                "@type": "ImageObject",
                url: "https://postgresus.com/logo.svg",
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
              <h1 id="notifiers">Notifiers</h1>

              <p className="text-lg text-gray-400">
                Postgresus supports multiple notification channels to keep you
                informed about your PostgreSQL backup status. Get instant alerts
                when backups succeed, fail or encounter issues.
              </p>

              <h2 id="supported-notifiers">Supported notifiers</h2>

              <ul>
                <li>
                  <Link
                    href="/notifiers/slack"
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Slack
                  </Link>{" "}
                  - Send notifications to Slack channels via webhooks
                </li>
                <li>
                  <strong>Discord</strong> - Post backup alerts to Discord
                  channels
                </li>
                <li>
                  <strong>Telegram</strong> - Receive notifications through
                  Telegram bots
                </li>
                <li>
                  <Link
                    href="/notifiers/teams"
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Microsoft Teams
                  </Link>{" "}
                  - Notify your team via Microsoft Teams channels
                </li>
                <li>
                  <strong>Email</strong> - Send email notifications for backup
                  events
                </li>
                <li>
                  <strong>Webhook</strong> - Custom webhook integration for any
                  service
                </li>
              </ul>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
