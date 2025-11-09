import type { Metadata } from "next";
import Image from "next/image";
import { CopyButton } from "./components/CopyButton";

export const metadata: Metadata = {
  title: "Postgresus | PostgreSQL backup",
  description:
    "Free and open source tool for PostgreSQL scheduled backups. Save them locally and to clouds. Notifications to Slack, Discord, etc.",
  keywords:
    "PostgreSQL, backup, monitoring, database, scheduled backups, Docker, self-hosted, open source, S3, Google Drive, Slack notifications, Discord, DevOps, database monitoring, pg_dump, database restore",
  robots: "index, follow",
  alternates: {
    canonical: "https://postgresus.com",
  },
  openGraph: {
    type: "website",
    url: "https://postgresus.com",
    title: "Postgresus | PostgreSQL backup",
    description:
      "Free and open source tool for PostgreSQL scheduled backups. Save them locally and to clouds. Notifications to Slack, Discord, etc.",
    images: [
      {
        url: "https://postgresus.com/images/index/dashboard.svg",
        alt: "Postgresus dashboard interface showing backup management",
      },
    ],
    siteName: "Postgresus",
    locale: "en_US",
  },
  twitter: {
    card: "summary_large_image",
    title: "Postgresus | PostgreSQL backup",
    description:
      "Free and open source tool for PostgreSQL scheduled backups. Save them locally and to clouds. Notifications to Slack, Discord, etc.",
    images: ["https://postgresus.com/images/index/dashboard.svg"],
  },
  applicationName: "Postgresus",
  appleWebApp: {
    title: "Postgresus",
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

export default function Index() {
  const installScript = `sudo apt-get install -y curl && \\
sudo curl -sSL https://raw.githubusercontent.com/RostislavDugin/postgresus/refs/heads/main/install-postgresus.sh | sudo bash`;

  const dockerRun = `docker run -d \\
  --name postgresus \\
  -p 4005:4005 \\
  -v ./postgresus-data:/postgresus-data \\
  --restart unless-stopped \\
  rostislavdugin/postgresus:latest`;

  const dockerCompose = `version: "3"

services:
  postgresus:
    container_name: postgresus
    image: rostislavdugin/postgresus:latest
    ports:
      - "4005:4005"
    volumes:
      - ./postgresus-data:/postgresus-data
    restart: unless-stopped`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "SoftwareApplication",
            name: "Postgresus",
            description:
              "Free and open source tool for PostgreSQL scheduled backups. Save them locally and to clouds. Notifications to Slack, Discord, etc.",
            url: "https://postgresus.com",
            image: "https://postgresus.com/images/index/dashboard.svg",
            logo: "https://postgresus.com/logo.svg",
            publisher: {
              "@type": "Organization",
              name: "Postgresus",
              logo: {
                "@type": "ImageObject",
                url: "https://postgresus.com/logo.svg",
              },
            },
            featureList: [
              "Scheduled PostgreSQL backups",
              "Multiple storage destinations (S3, Google Drive, Dropbox)",
              "Real-time notifications (Slack, Telegram, Discord)",
              "Database health monitoring",
              "Self-hosted via Docker",
              "Open source and free",
              "Support for PostgreSQL 13-17",
              "Backup compression and encryption",
            ],
            screenshot: "https://postgresus.com/images/index/dashboard.svg",
            softwareVersion: "latest",
          }),
        }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "Organization",
            name: "Postgresus",
            url: "https://postgresus.com/",
            alternateName: ["postgresus", "Postgresus"],
            logo: "https://postgresus.com/logo.svg",
            sameAs: ["https://github.com/RostislavDugin/postgresus"],
          }),
        }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "WebSite",
            name: "Postgresus",
            alternateName: ["postgresus", "PostgresUS"],
            url: "https://postgresus.com/",
            description: "PostgreSQL backup tool",
            publisher: { "@type": "Organization", name: "Postgresus" },
          }),
        }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "FAQPage",
            mainEntity: [
              {
                "@type": "Question",
                name: "What is Postgresus and why should I use it instead of hand-rolled scripts?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Postgresus is an Apache 2.0 licensed, self-hosted service backing up PostgreSQL, v13 to v18. It differs from shell scripts in that it has a frontend for scheduling tasks, compressing and storing archives on multiple targets (local disk, S3, Google Drive, Dropbox, etc.) and notifying your team when tasks finish or fail — all without hand-rolled code",
                },
              },
              {
                "@type": "Question",
                name: "How do I install Postgresus in the quickest manner?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "The most direct route is to run the one-line cURL installer. It fetches the current Docker image, spins up a single PostgreSQL container. Then creates a docker-compose.yml and boots up the service so it will automatically start again when reboots occur. Overall time is usually less than two minutes on a typical VPS.",
                },
              },
              {
                "@type": "Question",
                name: "What backup schedules can I schedule?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "You can choose from hourly, daily, weekly or monthly cycles and even choose an exact run time (such as 04:00 when it's late night). Weekly schedules enable you to choose a particular weekday, while monthly schedules enable you to choose a particular calendar day, giving you very fine-grained control of maintenance windows.",
                },
              },
              {
                "@type": "Question",
                name: "Where do my backups live and how much space will they occupy?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Archives can be saved to local volumes, S3-compatible buckets, Google Drive, Dropbox and other cloud targets. Postgresus implements balanced compression, which typically shrinks dump size by 4-8× with incremental only about 20% of runtime overhead, so you have storage and bandwidth savings.",
                },
              },
              {
                "@type": "Question",
                name: "How will I know a backup succeeded — or worse, failed?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Postgresus can notify with real-time emails, Slack, Telegram, webhooks, Mattermost, Discord and more. You have the choice of what channels to ping so that your DevOps team hears about successes and failures in real time, making recovery routines and compliance audits easier.",
                },
              },
              {
                "@type": "Question",
                name: "Does Postgresus reduce database security?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "No. All the data executes within containers you control, on servers you own. Credentials and backup files are left on your server or in the cloud account of your choice. Because it's open source, you or your security team, can inspect every line to make sure it meets your organization's needs before it's run.",
                },
              },
              {
                "@type": "Question",
                name: "Who is Postgresus suitable for?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Postgresus is designed for single developers, DevOps teams, organizations, startups, system administrators and IT departments who need reliable PostgreSQL backups. Whether you're managing personal projects or production databases, Postgresus provides enterprise-grade backup capabilities with a simple, intuitive interface.",
                },
              },
              {
                "@type": "Question",
                name: "How is Postgresus different from PgBackRest, Barman or pg_dump?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Postgresus provides a modern, user-friendly web interface instead of complex configuration files and command-line tools. While PgBackRest and Barman require extensive configuration and command-line expertise, Postgresus offers intuitive point-and-click setup. Unlike raw pg_dump scripts, it includes built-in scheduling, compression, multiple storage destinations, health monitoring and real-time notifications — all managed through a simple web UI.",
                },
              },
            ],
          }),
        }}
      />

      {/* NAVBAR */}
      <nav className="fixed top-0 z-50 flex h-[60px] w-full justify-center bg-white sm:h-[70px] md:h-[80px]">
        <div className="flex min-w-0 grow items-center border-b border-gray-200 px-4 sm:px-6 md:px-10">
          <Image
            src="/logo.svg"
            alt="Postgresus logo"
            width={35}
            height={35}
            className="h-auto w-[35px] shrink-0 sm:w-[40px] md:w-[50px]"
            priority
          />

          <div className="ml-2 select-none text-lg font-bold sm:ml-3 sm:text-xl md:ml-4 md:text-2xl">
            Postgresus
          </div>

          <div className="ml-auto mr-4 hidden gap-3 sm:mr-6 md:mr-10 lg:flex lg:gap-5">
            <a className="hover:opacity-70" href="#guide">
              Guide
            </a>
            <a className="hover:opacity-70" href="#installation">
              Installation
            </a>

            <a className="hover:opacity-70" href="/installation">
              Docs
            </a>
            <a
              className="hover:opacity-70"
              href="https://t.me/postgresus_community"
              target="_blank"
              rel="noopener noreferrer"
            >
              Community
            </a>
          </div>

          <a
            className="ml-auto lg:ml-0"
            href="https://github.com/RostislavDugin/postgresus"
            target="_blank"
            rel="noopener noreferrer"
          >
            <div className="flex items-center rounded-lg border border-gray-200 bg-[#f5f7f9] px-2 py-1 hover:bg-gray-100 md:px-4 md:py-2">
              <Image
                src="/images/index/github.svg"
                className="mr-1 h-auto w-4 sm:mr-2 md:mr-3"
                alt="GitHub icon"
                width={16}
                height={16}
                priority
              />
              <span className="text-sm sm:text-base">
                Star on GitHub
                <span className="hidden sm:inline">
                  , it&apos;s really important ❤️
                </span>
              </span>
            </div>
          </a>
        </div>
      </nav>
      {/* END OF NAVBAR */}

      {/* MAIN SECTION */}
      <div className="flex justify-center py-[50px] pt-[110px] md:py-[100px] md:pt-[180px]">
        <main className="w-[320px] max-w-[320px] sm:w-[640px] sm:max-w-[640px] lg:flex lg:w-[1200px] lg:max-w-[1200px]">
          <div className="lg:w-[500px] lg:min-w-[500px]">
            <h1 className="max-w-[300px] text-2xl font-bold sm:max-w-[400px] sm:text-3xl">
              PostgreSQL backup tool
            </h1>

            <p className="mt-5 max-w-[400px] sm:text-lg">
              Postgresus is a free, open source and self-hosted tool to backup
              PostgreSQL. Make backups with different storages and notifications
              about progress
            </p>

            <div className="mt-5 max-w-[280px] sm:text-lg md:max-w-full">
              <div className="mb-2 flex items-start gap-3">
                <Image
                  src="/images/index/check-blue.svg"
                  alt="Check mark icon"
                  className="mt-1 h-auto w-5"
                  width={20}
                  height={20}
                  priority
                />
                <p>Configurable health checks with notifications</p>
              </div>

              <div className="mb-2 flex items-start gap-3">
                <Image
                  src="/images/index/check-blue.svg"
                  alt="Check mark icon"
                  className="mt-1 h-auto w-5"
                  width={20}
                  height={20}
                />
                <p>
                  <b>Scheduled backups</b> (daily, weekly, at 4 AM, etc.)
                </p>
              </div>

              <div className="mb-2 flex items-start gap-3">
                <Image
                  src="/images/index/check-blue.svg"
                  alt="Check mark icon"
                  className="mt-1 h-auto w-5"
                  width={20}
                  height={20}
                />
                <p>Save backups locally, in S3, Google Drive and more</p>
              </div>

              <div className="mb-2 flex items-start gap-3">
                <Image
                  src="/images/index/check-blue.svg"
                  alt="Check mark icon"
                  className="mt-1 h-auto w-5"
                  width={20}
                  height={20}
                />
                <p>Notifications to Slack, Telegram, Discord, etc.</p>
              </div>

              <div className="mb-2 flex items-start gap-3">
                <Image
                  src="/images/index/check-blue.svg"
                  alt="Check mark icon"
                  className="mt-1 h-auto w-5"
                  width={20}
                  height={20}
                />
                <p>Run via .sh script, Docker or Docker Compose</p>
              </div>
            </div>

            <div className="mt-5 block text-sm sm:mt-10 sm:flex">
              <a
                href="#installation"
                className="block grow rounded-lg border-2 border-blue-600 bg-blue-600 px-4 py-2 text-center font-semibold text-white hover:bg-blue-700 sm:grow-0 sm:px-6 sm:py-3"
              >
                Configure for 2 minutes
              </a>

              <a
                href="https://github.com/RostislavDugin/postgresus"
                target="_blank"
                rel="noopener noreferrer"
                className="mt-1 block grow rounded-lg border-2 border-blue-600 px-4 py-2 text-center font-semibold text-blue-600 hover:bg-blue-100 sm:ml-2 sm:mt-0 sm:grow-0 sm:px-6 sm:py-3"
              >
                GitHub
              </a>
            </div>

            <div>
              <Image
                src="/images/index/messengers.svg"
                className="mt-5 h-auto w-[150px]"
                alt="Supported messengers and platforms"
                width={150}
                height={35}
                priority
              />
            </div>
          </div>

          <div className="mt-12 sm:mt-16 lg:mt-0 lg:grow">
            <Image
              className="h-auto w-full rounded"
              src="/images/index/dashboard.svg"
              alt="Postgresus dashboard interface"
              width={700}
              height={500}
              priority
            />
          </div>
        </main>
      </div>
      {/* END OF MAIN SECTION */}

      {/* HOW TO MAKE BACKUPS SECTION */}
      <div id="guide" className="flex justify-center py-[50px] md:py-[100px]">
        <div className="w-[320px] max-w-[320px] sm:w-[640px] sm:max-w-[640px] lg:w-[1200px] lg:max-w-[1200px]">
          <div className="text-center">
            <h2 className="text-2xl font-bold sm:text-4xl">
              How to make backups?
            </h2>
            <p className="mx-auto mt-5 max-w-[500px] text-base sm:text-lg">
              To make PostgreSQL backup with Postgresus you need to follow
              4&nbsp;steps. After that, you will be able to restore in one click
            </p>
          </div>

          <div className="mt-20">
            <div className="mb-16 block lg:flex">
              <div className="flex">
                <div className="mr-5 mt-1 flex">
                  <div className="flex h-6 w-6 items-center justify-center rounded-full bg-blue-600 text-xs font-bold text-white sm:h-9 sm:w-9 sm:text-base">
                    1
                  </div>
                </div>

                <div className="w-[500px] md:pr-20">
                  <h3 className="mb-3 text-lg font-bold md:text-2xl">
                    Select required schedule
                  </h3>

                  <p className="md:text-lg">
                    You can choose any time you need: daily, weekly, monthly and
                    particular time (like 4 AM)
                    <br />
                    <br />
                    For week interval you need to specify particular day, for
                    month you need to specify particular day
                    <br />
                    <br />
                    If your database is large, we recommend you choosing the
                    time when there are decrease in traffic
                  </p>
                </div>
              </div>

              <img
                src="/images/index/backup-step-1.webp"
                alt="Backup schedule configuration interface"
                className="mt-5 rounded-lg border border-gray-200 shadow-lg md:ml-14 md:mt-10 lg:ml-0 lg:mt-0"
                loading="lazy"
              />
            </div>

            <div className="mb-16 block lg:flex">
              <div className="flex">
                <div className="mr-5 mt-1 flex">
                  <div className="flex h-6 w-6 items-center justify-center rounded-full bg-blue-600 text-xs font-bold text-white sm:h-9 sm:w-9 sm:text-base">
                    2
                  </div>
                </div>

                <div className="w-[500px] md:pr-20">
                  <h3 className="mb-3 text-lg font-bold md:text-2xl">
                    Enter your database info
                  </h3>

                  <p className="md:text-lg">
                    Enter credentials of your PostgreSQL database, select
                    version and target DB. Also choose whether SSL is required
                    <br />
                    <br />
                    Postgresus, by default, compress backups at balance level to
                    not slow down backup process (~20% slower) and save x4-x8 of
                    the space (that decreasing network traffic)
                  </p>
                </div>
              </div>

              <img
                src="/images/index/backup-step-2.webp"
                alt="Database configuration form"
                className="mt-5 rounded-lg border border-gray-200 shadow-lg md:ml-14 md:mt-10 lg:ml-0 lg:mt-0"
                loading="lazy"
              />
            </div>

            <div className="mb-16 block lg:flex">
              <div className="flex">
                <div className="mr-5 mt-1 flex">
                  <div className="flex h-6 w-6 items-center justify-center rounded-full bg-blue-600 text-xs font-bold text-white sm:h-9 sm:w-9 sm:text-base">
                    3
                  </div>
                </div>

                <div className="w-[500px] md:pr-20">
                  <h3 className="mb-3 text-lg font-bold md:text-2xl">
                    Choose storage for backups
                  </h3>

                  <p className="md:text-lg">
                    You can keep files with backups locally, in S3, in Google
                    Drive, NAS, Dropbox and other services
                    <br />
                    <br />
                    Please keep in mind that you need to have enough space on
                    the storage
                  </p>
                </div>
              </div>

              <img
                src="/images/index/backup-step-3.webp"
                alt="Storage options selection interface"
                className="mt-5 rounded-lg border border-gray-200 shadow-lg md:ml-14 md:mt-10 lg:ml-0 lg:mt-0"
                loading="lazy"
              />
            </div>
          </div>

          <div className="mb-16 block lg:flex">
            <div className="flex">
              <div className="mr-5 mt-1 flex">
                <div className="flex h-6 w-6 items-center justify-center rounded-full bg-blue-600 text-xs font-bold text-white sm:h-9 sm:w-9 sm:text-base">
                  4
                </div>
              </div>

              <div className="w-[500px] md:pr-20">
                <h3 className="mb-3 text-lg font-bold md:text-2xl">
                  Choose where you want to receive notifications (optional)
                </h3>

                <p className="md:text-lg">
                  When backup succeed or failed, Postgresus is able to send you
                  notification. It can be chat with DevOps, your emails or even
                  webhook of your team
                  <br />
                  <br />
                  We are going to support the most of popular messangers and
                  platforms
                </p>
              </div>
            </div>

            <img
              src="/images/index/backup-step-4.webp"
              alt="Notification settings configuration"
              className="mt-5 rounded-lg border border-gray-200 shadow-lg md:ml-14 md:mt-10 lg:ml-0 lg:mt-0"
              loading="lazy"
            />
          </div>
        </div>
      </div>
      {/* END OF HOW TO MAKE BACKUPS SECTION */}

      {/* FEATURES SECTION */}
      <div
        id="features"
        className="flex justify-center bg-blue-100 py-[50px] md:py-[100px]"
      >
        <div className="space-b w-[320px] max-w-[320px] sm:w-[640px] sm:max-w-[640px] lg:w-[1200px] lg:max-w-[1200px]">
          <div className="mb-10 text-center md:mb-20">
            <h2 className="text-2xl font-bold sm:text-4xl">Features</h2>

            <p className="mx-auto mt-5 max-w-[550px] text-base sm:text-lg">
              Postgresus provides everything you need for reliable PostgreSQL
              backup management. From automated scheduling to multiple storage
              options, we&apos;ve got you covered.
            </p>
          </div>

          <div className="flex flex-wrap gap-3">
            <div className="h-[320px] max-h-[320px] w-[320px] rounded-2xl bg-white p-8 md:h-[390px] md:max-h-[390px] md:w-[390px]">
              <h3 className="mb-3 text-xl font-bold md:text-2xl">
                Scheduled backups
              </h3>

              <div className="h-[120px] max-h-[120px] sm:h-[220px] sm:max-h-[220px]">
                <img
                  src="/images/index/feature-scheduled-backups.webp"
                  className="max-h-[120px] sm:max-h-[220px]"
                  alt="Scheduled backups"
                  loading="lazy"
                />
              </div>

              <div className="mt-3 text-sm text-gray-600 md:mt-0 md:text-base">
                Backup is a thing that should be done in specified time on
                regular basis. So we provide many options: from hourly to
                monthly
              </div>
            </div>

            <div className="h-[320px] max-h-[320px] w-[320px] rounded-2xl bg-white p-8 md:h-[390px] md:max-h-[390px] md:w-[390px]">
              <h3 className="mb-3 text-xl font-bold md:text-2xl">
                Notifications
              </h3>

              <div className="flex h-[120px] max-h-[120px] justify-center pt-7 sm:h-[200px] sm:max-h-[200px]">
                <img
                  className="max-h-[120px] sm:max-h-[150px]"
                  src="/images/index/feature-notifications.svg"
                  alt="Notifications"
                  loading="lazy"
                />
              </div>

              <div className="pt-5 text-sm text-gray-600 sm:pt-5 sm:text-base">
                You can receive notifications about success or fail of the
                process. This is useful for developers or DevOps teams
              </div>
            </div>

            <div className="h-[320px] max-h-[320px] w-[320px] rounded-2xl bg-white p-8 md:h-[390px] md:max-h-[390px] md:w-[390px]">
              <h3 className="mb-3 text-xl font-bold md:text-2xl">
                Self hosted via Docker
              </h3>

              <div className="flex h-[140px] max-h-[140px] justify-center pt-5 sm:mt-0 sm:h-[220px] sm:max-h-[220px]">
                <img
                  className="max-h-[120px] sm:max-h-[150px]"
                  src="/images/index/feature-docker.webp"
                  alt="Self hosted via Docker"
                  loading="lazy"
                />
              </div>

              <div className="mt-3 text-sm text-gray-600 md:mt-0 md:text-base">
                Postgresus runs on your PC or VPS. Therefore, all your data is
                owned by you and secured. Deploy takes about 2 minutes
              </div>
            </div>

            <div className="h-[320px] max-h-[320px] w-[320px] rounded-2xl bg-white p-8 md:h-[390px] md:max-h-[390px] md:w-[390px]">
              <h3 className="mb-3 text-xl font-bold md:text-2xl">
                Configurable health checks
              </h3>

              <div className="h-[50px] max-h-[50px] sm:h-[120px] sm:max-h-[120px]">
                <img
                  className="max-h-[50px] sm:max-h-[120px]"
                  src="/images/index/feature-healthcheck.svg"
                  alt="Configurable health checks"
                  loading="lazy"
                />
              </div>

              <div className="mt-3 text-sm text-gray-600 md:mt-0 md:text-base">
                Each minute (or any another amount of time) the system will ping
                your database and show you the history of attempts
                <br />
                <br />
                The database can be considered as down after 3 failed attempts
                or so. Once DB is healthy again - you receive notification too
              </div>
            </div>

            <div className="h-[320px] max-h-[320px] w-[320px] rounded-2xl bg-white p-8 md:h-[390px] md:max-h-[390px] md:w-[390px]">
              <h3 className="mb-3 text-xl font-bold md:text-2xl">
                Open source and free
              </h3>

              <div className="flex h-[90px] max-h-[90px] justify-center pt-10 sm:my-0 sm:h-[160px] sm:max-h-[160px]">
                <img
                  className="max-h-[90px] sm:max-h-[100px]"
                  src="/images/index/feature-github.svg"
                  alt="Open source and free"
                  loading="lazy"
                />
              </div>

              <div className="pt-10 text-sm text-gray-600 md:mt-0 md:text-base">
                The project is fully open source, free and have Apache 2.0
                license. You can copy and fork the code on your own
              </div>
            </div>

            <div className="h-[320px] max-h-[320px] w-[320px] rounded-2xl bg-white p-8 md:h-[390px] md:max-h-[390px] md:w-[390px]">
              <h3 className="mb-3 text-xl font-bold md:text-2xl">
                Many PostgreSQL versions
              </h3>

              <div className="flex h-[90px] max-h-[90px] justify-center pt-10 sm:my-0 sm:h-[160px] sm:max-h-[160px]">
                <img
                  className="max-h-[90px] sm:max-h-[100px]"
                  src="/images/index/feature-postgresql.svg"
                  alt="Many PostgreSQL versions"
                  loading="lazy"
                />
              </div>

              <div className="pt-10 text-sm text-gray-600 md:mt-0 md:text-base">
                PostgreSQL 13, 14, 15, 16, 17 and 18 are supported by the
                project. You can backup any version from 2020
              </div>
            </div>

            <div className="h-[320px] max-h-[320px] w-[320px] rounded-2xl bg-white p-8 md:h-[390px] md:max-h-[390px] md:w-[390px]">
              <h3 className="mb-3 text-xl font-bold md:text-2xl">
                Many destinations to store
              </h3>

              <div className="flex h-[120px] max-h-[120px] justify-center pb-5 pt-5 sm:my-0 sm:h-[200px] sm:max-h-[200px] sm:pb-0 sm:pt-10">
                <img
                  className="max-h-[120px] sm:max-h-[120px]"
                  src="/images/index/feature-google-drive.svg"
                  alt="Many destinations to store"
                  loading="lazy"
                />
              </div>

              <div className="mt-3 text-sm text-gray-600 md:mt-0 md:text-base">
                Files are kept on VPS, cloud storages and other places. You can
                choose any storage you. Files are always owned by you need
              </div>
            </div>
          </div>
        </div>
      </div>
      {/* END OF FEATURES SECTION */}

      {/* INSTALLATION SECTION */}
      <div
        id="installation"
        className="flex justify-center py-[50px] md:py-[100px]"
      >
        <div className="w-[320px] max-w-[320px] sm:w-[640px] sm:max-w-[640px] lg:w-[1200px] lg:max-w-[1200px]">
          <div className="mb-10 md:mb-20">
            <h2 className="text-2xl font-bold sm:text-4xl">How to install?</h2>

            <p className="mt-5 max-w-[550px] text-base sm:text-lg">
              You have three ways to install Postgresus: automated script
              (recommended), simple Docker run or Docker Compose setup.
            </p>
          </div>

          <div className="mt-20">
            <div className="mb-16 block xl:flex">
              <div className="w-full sm:pr-10 xl:w-1/2">
                <h3 className="mb-3 text-xl font-bold md:text-2xl">
                  Option 1: automated installation script (recommended)
                </h3>

                <p className="max-w-[500px] md:text-lg">
                  The installation script will:
                  <br />
                  ✅ Install Docker with Docker Compose (if not already
                  installed)
                  <br />
                  ✅ Set up Postgresus
                  <br />✅ Configure automatic startup on system reboot
                </p>

                <div className="relative mt-5 max-w-[550px]">
                  <code className="block w-full overflow-x-auto whitespace-nowrap rounded-lg bg-gray-100 p-4 pr-16 text-sm">
                    sudo apt-get install -y curl && \
                    <br />
                    sudo curl -sSL
                    https://raw.githubusercontent.com/RostislavDugin/postgresus/refs/heads/main/install-postgresus.sh
                    \ | sudo bash
                  </code>
                  <div className="absolute right-2 top-2">
                    <CopyButton text={installScript} />
                  </div>
                </div>
              </div>

              <div className="mt-20 w-full sm:pr-10 xl:mt-0 xl:w-1/2">
                <h3 className="mb-3 text-xl font-bold md:text-2xl">
                  Option 2: simple Docker run
                </h3>

                <p className="max-w-[500px] md:text-lg">
                  The easiest way to run Postgresus with embedded PostgreSQL.
                  This single command will:
                  <br />
                  ✅ Start Postgresus
                  <br />
                  ✅ Store all data in ./postgresus-data directory
                  <br />✅ Automatically restart on system reboot
                </p>

                <div className="relative mt-5 max-w-[550px]">
                  <code className="block w-full overflow-x-auto whitespace-nowrap rounded-lg bg-gray-100 p-4 pr-16 text-sm">
                    docker run -d \
                    <br />
                    &nbsp;&nbsp;--name postgresus \
                    <br />
                    &nbsp;&nbsp;-p 4005:4005 \
                    <br />
                    &nbsp;&nbsp;-v ./postgresus-data:/postgresus-data \
                    <br />
                    &nbsp;&nbsp;--restart unless-stopped \
                    <br />
                    &nbsp;&nbsp;rostislavdugin/postgresus:latest
                  </code>
                  <div className="absolute right-2 top-2">
                    <CopyButton text={dockerRun} />
                  </div>
                </div>
              </div>
            </div>

            <div className="block xl:flex">
              <div className="w-full sm:pr-10">
                <h3 className="mb-3 text-xl font-bold md:text-2xl">
                  Option 3: Docker Compose setup
                </h3>

                <p className="max-w-[500px] md:text-lg">
                  Create a docker-compose.yml file with the following
                  configuration, then run: <strong>docker compose up -d</strong>
                </p>

                <div className="relative mt-5 max-w-[550px]">
                  <code className="block w-full overflow-x-auto whitespace-nowrap rounded-lg bg-gray-100 p-4 pr-16 text-sm">
                    version: &quot;3&quot;
                    <br />
                    <br />
                    services:
                    <br />
                    &nbsp;&nbsp;postgresus:
                    <br />
                    &nbsp;&nbsp;&nbsp;&nbsp;container_name: postgresus
                    <br />
                    &nbsp;&nbsp;&nbsp;&nbsp;image:
                    rostislavdugin/postgresus:latest
                    <br />
                    &nbsp;&nbsp;&nbsp;&nbsp;ports:
                    <br />
                    &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;- &quot;4005:4005&quot;
                    <br />
                    &nbsp;&nbsp;&nbsp;&nbsp;volumes:
                    <br />
                    &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;-
                    ./postgresus-data:/postgresus-data
                    <br />
                    &nbsp;&nbsp;&nbsp;&nbsp;restart: unless-stopped
                  </code>

                  <div className="absolute right-2 top-2">
                    <CopyButton text={dockerCompose} />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      {/* END OF INSTALLATION SECTION */}

      {/* FAQ SECTION */}
      <div
        id="faq"
        className="flex justify-center bg-blue-100 py-[50px] md:py-[100px]"
      >
        <div className="w-[320px] max-w-[320px] sm:w-[640px] sm:max-w-[640px] lg:w-[1200px] lg:max-w-[1200px]">
          <div className="mb-10 md:mb-20">
            <h2 className="text-2xl font-bold sm:text-4xl">FAQ</h2>
            <p className="mt-5 max-w-[600px] sm:text-lg">
              The goal of Postgresus — make backing up as simple as possible for
              single developers (as well as DevOps) and teams. UI makes it easy
              to create backups and visualizes the progress and restores
              anything in couple of clicks
            </p>
          </div>

          <div className="mt-15">
            <div className="flex flex-wrap">
              <div className="mb-8 w-full pr-10 lg:w-1/2">
                <h3 className="mb-3 max-w-[350px] font-bold md:text-xl">
                  1. What is Postgresus and why should I use it instead of
                  hand-rolled scripts?
                </h3>

                <p className="max-w-[500px] md:text-lg">
                  Postgresus is an Apache 2.0 licensed, self-hosted service
                  backing up PostgreSQL, v13 to v18. It differs from shell
                  scripts in that it has a frontend for scheduling tasks,
                  compressing and storing archives on multiple targets (local
                  disk, S3, Google Drive, NAS, Dropbox, etc.) and notifying your
                  team when tasks finish or fail — all without hand-rolled code
                </p>
              </div>

              <div className="mb-8 w-full pr-10 lg:w-1/2">
                <h3 className="mb-3 max-w-[350px] font-bold md:text-xl">
                  2. How do I install Postgresus in the quickest manner?
                </h3>

                <p className="max-w-[500px] md:text-lg">
                  The most direct route is to run the one-line cURL installer.
                  It fetches the current Docker image, spins up a single
                  PostgreSQL container. Then creates a docker-compose.yml and
                  boots up the service so it will automatically start again when
                  reboots occur. Overall time is usually less than two minutes
                  on a typical VPS.
                </p>
              </div>

              <div className="mb-8 w-full pr-10 lg:w-1/2">
                <h3 className="mb-3 max-w-[350px] font-bold md:text-xl">
                  3. What backup schedules can I schedule?
                </h3>

                <p className="max-w-[500px] md:text-lg">
                  You can choose from hourly, daily, weekly or monthly cycles
                  and even choose an exact run time (such as 04:00 when
                  it&apos;s late night). Weekly schedules enable you to choose a
                  particular weekday, while monthly schedules enable you to
                  choose a particular calendar day, giving you very fine-grained
                  control of maintenance windows.
                </p>
              </div>

              <div className="mb-8 w-full pr-10 lg:w-1/2">
                <h3 className="mb-3 max-w-[350px] font-bold md:text-xl">
                  4. Where do my backups live and how much space will they
                  occupy?
                </h3>

                <p className="max-w-[500px] md:text-lg">
                  Archives can be saved to local volumes, S3-compatible buckets,
                  Google Drive, Dropbox and other cloud targets. Postgresus
                  implements balanced compression, which typically shrinks dump
                  size by 4-8x with incremental only about 20% of runtime
                  overhead, so you have storage and bandwidth savings.
                </p>
              </div>

              <div className="mb-8 w-full pr-10 lg:w-1/2">
                <h3 className="mb-3 max-w-[350px] font-bold md:text-xl">
                  5. How will I know a backup succeeded — or worse, failed?
                </h3>

                <p className="max-w-[500px] md:text-lg">
                  Postgresus can notify with real-time emails, Slack, Telegram,
                  webhooks, Mattermost, Discord and more. You have the choice of
                  what channels to ping so that your DevOps team hears about
                  successes and failures in real time, making recovery routines
                  and compliance audits easier.
                </p>
              </div>

              <div className="mb-8 w-full pr-10 lg:w-1/2">
                <h3 className="mb-3 max-w-[350px] font-bold md:text-xl">
                  6. Does Postgresus reduce database security?
                </h3>

                <p className="max-w-[500px] md:text-lg">
                  No. All the data executes within containers you control, on
                  servers you own. Credentials and backup files are left on your
                  server or in the cloud account of your choice. Because
                  it&apos;s open source, you or your security team, can inspect
                  every line to make sure it meets your organization&apos;s
                  needs before it&apos;s run.
                </p>
              </div>

              <div className="mb-8 w-full pr-10 lg:w-1/2">
                <h3 className="mb-3 max-w-[350px] font-bold md:text-xl">
                  7. How do I set up and run my first backup job in Postgresus?
                </h3>

                <p className="max-w-[500px] md:text-lg">
                  To start your very first Postgresus backup, simply log in to
                  the dashboard, click on New Backup, select an interval —
                  hourly, daily, weekly or monthly. Then specify the exact run
                  time (e.g., 02:30 for off-peak hours). Then input your
                  PostgreSQL host, port number, database name, credentials,
                  version and SSL preference. Choose where the archive should be
                  sent (local path, S3 bucket, Google Drive folder, Dropbox,
                  etc.). If you need, add notification channels such as email,
                  Slack, Telegram or a webhook and click Save. Postgresus
                  instantly validates the info, starts the schedule, runs the
                  initial job and sends live status. So you may restore with one
                  touch when the backup is complete.
                </p>
              </div>

              <div className="mb-8 w-full pr-10 lg:w-1/2">
                <h3 className="mb-3 max-w-[350px] font-bold md:text-xl">
                  8. How does PostgreSQL monitoring work?
                </h3>

                <p className="max-w-[500px] md:text-lg">
                  Postgresus monitors your databases instantly. This optional
                  feature helps avoid extra costs for edge DBs. Health checks
                  are performed once a specific period (minute, 5 minutes,
                  etc.). To enable the feature, choose your DB and select
                  &quot;enable&quot; monitoring. Then configure health checks
                  period and number of failed attempts to consider the DB as
                  unavailable.
                </p>
              </div>

              <div className="mb-8 w-full pr-10 lg:w-1/2">
                <h3 className="mb-3 max-w-[350px] font-bold md:text-xl">
                  9. Who is Postgresus suitable for?
                </h3>

                <p className="max-w-[500px] md:text-lg">
                  Postgresus is designed for single developers, DevOps teams,
                  organizations, startups, system administrators and IT
                  departments who need reliable PostgreSQL backups. Whether
                  you&apos;re managing personal projects or production
                  databases, Postgresus provides enterprise-grade backup
                  capabilities with a simple, intuitive interface.
                </p>
              </div>

              <div className="mb-8 w-full pr-10 lg:w-1/2">
                <h3 className="mb-3 max-w-[350px] font-bold md:text-xl">
                  10. How is Postgresus different from PgBackRest, Barman or
                  pg_dump?
                </h3>

                <p className="max-w-[500px] md:text-lg">
                  Postgresus provides a modern, user-friendly web interface
                  instead of complex configuration files and command-line tools.
                  While PgBackRest and Barman require extensive configuration
                  and command-line expertise, Postgresus offers intuitive
                  point-and-click setup. Unlike raw pg_dump scripts, it includes
                  built-in scheduling, compression, multiple storage
                  destinations, health monitoring and real-time notifications —
                  all managed through a simple web UI.
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
      {/* END OF FAQ SECTION */}

      {/* FOOTER SECTION */}
      <div
        id="footer"
        className="flex justify-center bg-blue-600 py-[50px] md:py-[50px]"
      >
        <div className="w-[320px] max-w-[320px] sm:w-[640px] sm:max-w-[640px] lg:w-[1200px] lg:max-w-[1200px]">
          <div className="flex flex-col items-center space-y-4">
            <div className="flex flex-wrap justify-center gap-6 text-white">
              <a
                href="/installation"
                className="transition-colors hover:text-blue-200"
              >
                Documentation
              </a>

              <a
                href="https://github.com/RostislavDugin/postgresus"
                target="_blank"
                rel="noopener noreferrer"
                className="transition-colors hover:text-blue-200"
              >
                GitHub
              </a>

              <a
                href="https://t.me/postgresus_community"
                target="_blank"
                rel="noopener noreferrer"
                className="transition-colors hover:text-blue-200"
              >
                Community
              </a>

              <a
                href="https://rostislav-dugin.com"
                target="_blank"
                rel="noopener noreferrer"
                className="transition-colors hover:text-blue-200"
              >
                Developer
              </a>
            </div>
            <p className="text-center text-sm text-white">
              &copy; 2025 Postgresus. All rights reserved.
            </p>
          </div>
        </div>
      </div>
      {/* END OF FOOTER SECTION */}
    </>
  );
}
