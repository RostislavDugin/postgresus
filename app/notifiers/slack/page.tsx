import type { Metadata } from "next";
import DocsNavbarComponent from "../../components/DocsNavbarComponent";
import DocsSidebarComponent from "../../components/DocsSidebarComponent";
import DocTableOfContentComponent from "../../components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title: "How to configure Slack notifications for Postgresus | Postgresus",
  description:
    "Step-by-step guide to set up Slack notifications for PostgreSQL backup alerts with Postgresus. Learn how to create Slack webhook and configure notifications.",
  keywords: [
    "Postgresus",
    "Slack notifications",
    "PostgreSQL backup",
    "Slack webhook",
    "backup alerts",
    "database notifications",
  ],
  openGraph: {
    title: "How to configure Slack notifications for Postgresus | Postgresus",
    description:
      "Step-by-step guide to set up Slack notifications for PostgreSQL backup alerts with Postgresus. Learn how to create Slack webhook and configure notifications.",
    type: "article",
    url: "https://postgresus.com/notifiers/slack",
  },
  twitter: {
    card: "summary",
    title: "How to configure Slack notifications for Postgresus | Postgresus",
    description:
      "Step-by-step guide to set up Slack notifications for PostgreSQL backup alerts with Postgresus. Learn how to create Slack webhook and configure notifications.",
  },
  alternates: {
    canonical: "https://postgresus.com/notifiers/slack",
  },
  robots: "index, follow",
};

export default function SlackPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "How to configure Slack notifications for Postgresus",
            description:
              "Step-by-step guide to set up Slack notifications for PostgreSQL backup alerts with Postgresus",
            step: [
              {
                "@type": "HowToStep",
                name: "Open Slack workspace",
                text: "Navigate to your Slack workspace where you want to receive notifications.",
              },
              {
                "@type": "HowToStep",
                name: "Access Slack apps",
                text: "Go to your Slack workspace and access the Apps section.",
              },
              {
                "@type": "HowToStep",
                name: "Create incoming webhook",
                text: "Search for and add the Incoming Webhooks app to your workspace.",
              },
              {
                "@type": "HowToStep",
                name: "Select channel",
                text: "Choose the channel where you want to receive backup notifications.",
              },
              {
                "@type": "HowToStep",
                name: "Copy webhook URL",
                text: "Copy the generated webhook URL from Slack.",
              },
              {
                "@type": "HowToStep",
                name: "Configure in Postgresus",
                text: "Paste the webhook URL into Postgresus notifier configuration.",
              },
              {
                "@type": "HowToStep",
                name: "Test the notification",
                text: "Test the notification to ensure it's working correctly.",
              },
            ],
          }),
        }}
      />

      <DocsNavbarComponent />

      <div className="flex min-h-screen">
        {/* Sidebar */}
        <DocsSidebarComponent />

        {/* Main Content */}
        <main className="flex-1 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="slack-notifications">Slack notifications</h1>

              <p className="text-lg text-gray-700">
                Configure Slack to receive instant notifications about your
                PostgreSQL backup status. Get alerts for successful backups,
                failures and warnings directly in your Slack channels.
              </p>

              <h2 id="setup-slack-webhook">Setup Slack webhook</h2>

              <h3 id="open-slack-workspace">1. Open your Slack workspace</h3>

              <p>
                Navigate to your Slack workspace where you want to receive
                backup notifications.
              </p>

              <Image
                src="/images/notifier-slack/image-1.png"
                alt="Open Slack workspace"
                width={800}
                height={500}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="access-slack-apps">2. Access Slack apps</h3>

              <p>
                Click on your workspace name in the top left, then select{" "}
                <strong>&quot;Settings &amp; administration&quot;</strong> →{" "}
                <strong>&quot;Manage apps&quot;</strong>.
              </p>

              <Image
                src="/images/notifier-slack/image-2.png"
                alt="Access Slack Apps"
                width={600}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="search-incoming-webhooks">
                3. Search for incoming webhooks
              </h3>

              <p>
                In the App Directory, search for{" "}
                <strong>&quot;Incoming Webhooks&quot;</strong> and click on it.
              </p>

              <Image
                src="/images/notifier-slack/image-3.png"
                alt="Search for Incoming Webhooks"
                width={700}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="add-to-slack">4. Add to Slack</h3>

              <p>
                Click the <strong>&quot;Add to Slack&quot;</strong> button to
                install the Incoming Webhooks app to your workspace.
              </p>

              <Image
                src="/images/notifier-slack/image-4.png"
                alt="Add to Slack"
                width={700}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="select-channel">5. Select channel</h3>

              <p>
                Choose the channel where you want to receive backup
                notifications, then click{" "}
                <strong>&quot;Add Incoming Webhooks integration&quot;</strong>.
              </p>

              <Image
                src="/images/notifier-slack/image-5.png"
                alt="Select channel for notifications"
                width={600}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="copy-webhook-url">6. Copy webhook URL</h3>

              <p>
                After creating the webhook, you&apos;ll see the{" "}
                <strong>Webhook URL</strong>. Copy this URL - you&apos;ll need
                it for Postgresus configuration.
              </p>

              <Image
                src="/images/notifier-slack/image-6.png"
                alt="Copy webhook URL"
                width={700}
                height={500}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="configure-postgresus">Configure in Postgresus</h2>

              <h3 id="add-slack-notifier">1. Add Slack notifier</h3>

              <p>
                In Postgresus, navigate to the notifiers settings and add a new
                Slack notifier. Paste the webhook URL you copied from Slack.
              </p>

              <Image
                src="/images/notifier-slack/image-7.png"
                alt="Configure Slack in Postgresus"
                width={600}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="test-notification">2. Test the notification</h3>

              <p>
                After configuring the webhook, test the notification to ensure
                it&apos;s working correctly. You should receive a test message
                in your selected Slack channel.
              </p>

              <p>
                That&apos;s it! Your Slack workspace is now configured to
                receive PostgreSQL backup notifications from Postgresus.
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/notifiers"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Back to notifiers
                </a>
              </div>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
