import type { Metadata } from "next";
import Link from "next/link";
import InstallationComponent from "./components/InstallationComponent";
import LiteYouTubeEmbed from "./components/LiteYouTubeEmbed";

export const metadata: Metadata = {
  title: "Postgresus | PostgreSQL backup",
  description:
    "Free and open source tool for PostgreSQL scheduled backups. Save them locally and to clouds. Notifications to Slack, Discord, Telegram, email, webhook, etc.",
  keywords:
    "PostgreSQL, backup, monitoring, database, scheduled backups, Docker, self-hosted, open source, S3, Google Drive, Slack notifications, Discord, DevOps, database monitoring, pg_dump, database restore, encryption, AES-256, backup encryption",
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
        url: "https://postgresus.com/images/index/dashboard.png",
        alt: "Postgresus dashboard interface showing backup management",
        width: 1735,
        height: 850,
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
    images: ["https://postgresus.com/images/index/dashboard.png"],
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
  return (
    <div className="overflow-x-hidden">
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
            image: "https://postgresus.com/images/index/dashboard.png",
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
              "Multiple storage destinations (S3, Google Drive, Dropbox, etc.)",
              "Real-time notifications (Slack, Telegram, Discord, Webhook, email, etc.)",
              "Database health monitoring",
              "Self-hosted via Docker",
              "Open source and free",
              "Support for PostgreSQL 12-18",
              "Backup compression and AES-256-GCM encryption",
            ],
            screenshot: "https://postgresus.com/images/index/dashboard.png",
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
                name: "How does Postgresus ensure security?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Postgresus enforces security on three levels: (1) Sensitive data encryption — all passwords, tokens and credentials are encrypted with AES-256-GCM and stored separately from the database; (2) Backup encryption — each backup file is encrypted with a unique key derived from a master key, backup ID and random salt, making backups useless without your encryption key even if someone gains storage access; (3) Read-only database access — Postgresus only requires SELECT permissions and performs comprehensive checks to ensure no write privileges exist, preventing data corruption even if the tool is compromised. All operations run in containers you control on servers you own, and because it's open source, your security team can audit every line of code before deployment.",
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

      {/* HEADER */}
      <header className="fixed top-0 left-0 right-0 z-50 flex justify-center pt-3 md:pt-5 px-4 md:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <nav className="relative flex items-center justify-between border backdrop-blur-md bg-[#0C0E13]/80 md:bg-[#0C0E13]/20 border-[#ffffff20] px-3 py-2 rounded-xl">
            <Link href="/" className="flex items-center gap-2.5">
              <img
                src="/logo.svg"
                alt="Postgresus logo"
                width={32}
                height={32}
                className="h-7 w-7 md:h-8 md:w-8"
              />

              <span className="text-base md:text-lg font-semibold">
                Postgresus
              </span>
            </Link>

            {/* Desktop Navigation */}
            <div className="absolute left-1/2 -translate-x-1/2 hidden lg:flex items-center gap-3">
              <a
                href="#how-to-use"
                className="py-2 hover:text-gray-300 transition-colors"
              >
                How to use
              </a>

              <a
                href="#features"
                className="py-2 hover:text-gray-300 transition-colors"
              >
                Features
              </a>

              <a
                href="/installation"
                className="py-2 hover:text-gray-300 transition-colors"
              >
                Docs
              </a>
              <a
                href="/contribute"
                className="py-2 hover:text-gray-300 transition-colors"
              >
                Contribute
              </a>
              <a
                href="https://t.me/postgresus_community"
                target="_blank"
                rel="noopener noreferrer"
                className="py-2 hover:text-gray-300 transition-colors"
              >
                Community
              </a>
            </div>

            {/* GitHub Button */}
            <a
              href="https://github.com/RostislavDugin/postgresus"
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
                <g clipPath="url(#clip0_1_2459)">
                  <path
                    fillRule="evenodd"
                    clipRule="evenodd"
                    d="M9.9702 0C4.45694 0 0 4.4898 0 10.0443C0 14.4843 2.85571 18.2427 6.81735 19.5729C7.31265 19.6729 7.49408 19.3567 7.49408 19.0908C7.49408 18.858 7.47775 18.0598 7.47775 17.2282C4.70429 17.8269 4.12673 16.0308 4.12673 16.0308C3.68102 14.8667 3.02061 14.5676 3.02061 14.5676C2.11286 13.9522 3.08673 13.9522 3.08673 13.9522C4.09367 14.0188 4.62204 14.9833 4.62204 14.9833C5.51327 16.5131 6.94939 16.0808 7.52714 15.8147C7.60959 15.1661 7.87388 14.7171 8.15449 14.4678C5.94245 14.2349 3.6151 13.3702 3.6151 9.51204C3.6151 8.41449 4.01102 7.51653 4.63837 6.81816C4.53939 6.56878 4.19265 5.53755 4.73755 4.15735C4.73755 4.15735 5.57939 3.89122 7.47755 5.18837C8.29022 4.9685 9.12832 4.85666 9.9702 4.85571C10.812 4.85571 11.6702 4.97225 12.4627 5.18837C14.361 3.89122 15.2029 4.15735 15.2029 4.15735C15.7478 5.53755 15.4008 6.56878 15.3018 6.81816C15.9457 7.51653 16.3253 8.41449 16.3253 9.51204C16.3253 13.3702 13.998 14.2182 11.7694 14.4678C12.1327 14.7837 12.4461 15.3822 12.4461 16.3302C12.4461 17.6771 12.4298 18.7582 12.4298 19.0906C12.4298 19.3567 12.6114 19.6729 13.1065 19.5731C17.0682 18.2424 19.9239 14.4843 19.9239 10.0443C19.9402 4.4898 15.4669 0 9.9702 0Z"
                    fill="white"
                  />
                </g>
                <defs>
                  <clipPath id="clip0_1_2459">
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

      {/* MAIN SECTION */}
      <main className="relative overflow-hidden pt-[60px] md:pt-[68px]">
        <div className="relative mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px] px-4 md:px-6 lg:px-0 pt-12 md:pt-[100px] pb-12 md:pb-[100px]">
          {/* Background ellipse */}
          <div className="relative">
            <div className="absolute left-1/2 -translate-x-1/2 w-[300px] md:w-[480px] h-[300px] md:h-[480px] bg-[#155dfc]/7 top-[-50px] md:top-[-100px] rounded-full blur-3xl -z-10" />
          </div>

          {/* Content */}
          <div className="text-center mb-8 md:mb-16">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">Postgresus</span>
            </div>

            <h1 className="text-2xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6 mx-auto sm:max-w-none">
              PostgreSQL backup tool
            </h1>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-6 md:mb-10 px-2">
              Postgresus is a free, open source and self-hosted tool to backup
              PostgreSQL. Make backups with different storages (S3, Google
              Drive, FTP, etc.) and notifications about progress (Slack,
              Discord, Telegram, etc.)
            </p>

            <div className="flex flex-col sm:flex-row items-center justify-center gap-2 sm:gap-2">
              <a
                href="#installation"
                className="w-full sm:w-auto inline-flex items-center justify-center gap-2 px-5 py-2.5 bg-white rounded-lg text-black font-medium hover:opacity-70 transition-opacity"
              >
                <span>Configure in 2 minutes</span>
                <svg
                  aria-hidden={true}
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <path d="M5 12h14M12 5l7 7-7 7" />
                </svg>
              </a>

              <a
                href="https://github.com/RostislavDugin/postgresus"
                target="_blank"
                rel="noopener noreferrer"
                className="w-full sm:w-auto inline-flex items-center justify-center gap-2 px-5 py-2.5 rounded-lg font-medium border border-[#ffffff20] bg-[#0C0E13] hover:opacity-70 transition-opacity"
              >
                <svg
                  aria-hidden={true}
                  width="24"
                  height="24"
                  viewBox="0 0 20 20"
                  fill="none"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <g clipPath="url(#clip0_1_2459)">
                    <path
                      fillRule="evenodd"
                      clipRule="evenodd"
                      d="M9.9702 0C4.45694 0 0 4.4898 0 10.0443C0 14.4843 2.85571 18.2427 6.81735 19.5729C7.31265 19.6729 7.49408 19.3567 7.49408 19.0908C7.49408 18.858 7.47775 18.0598 7.47775 17.2282C4.70429 17.8269 4.12673 16.0308 4.12673 16.0308C3.68102 14.8667 3.02061 14.5676 3.02061 14.5676C2.11286 13.9522 3.08673 13.9522 3.08673 13.9522C4.09367 14.0188 4.62204 14.9833 4.62204 14.9833C5.51327 16.5131 6.94939 16.0808 7.52714 15.8147C7.60959 15.1661 7.87388 14.7171 8.15449 14.4678C5.94245 14.2349 3.6151 13.3702 3.6151 9.51204C3.6151 8.41449 4.01102 7.51653 4.63837 6.81816C4.53939 6.56878 4.19265 5.53755 4.73755 4.15735C4.73755 4.15735 5.57939 3.89122 7.47755 5.18837C8.29022 4.9685 9.12832 4.85666 9.9702 4.85571C10.812 4.85571 11.6702 4.97225 12.4627 5.18837C14.361 3.89122 15.2029 4.15735 15.2029 4.15735C15.7478 5.53755 15.4008 6.56878 15.3018 6.81816C15.9457 7.51653 16.3253 8.41449 16.3253 9.51204C16.3253 13.3702 13.998 14.2182 11.7694 14.4678C12.1327 14.7837 12.4461 15.3822 12.4461 16.3302C12.4461 17.6771 12.4298 18.7582 12.4298 19.0906C12.4298 19.3567 12.6114 19.6729 13.1065 19.5731C17.0682 18.2424 19.9239 14.4843 19.9239 10.0443C19.9402 4.4898 15.4669 0 9.9702 0Z"
                      fill="white"
                    />
                  </g>
                  <defs>
                    <clipPath id="clip0_1_2459">
                      <rect width="20" height="20" fill="white" />
                    </clipPath>
                  </defs>
                </svg>

                <span>GitHub</span>
              </a>
            </div>
          </div>

          {/* Dashboard Screenshot */}
          <div className="relative mx-auto max-w-[1200px]">
            <div>
              <img
                src="/images/index/dashboard.svg"
                alt="Postgresus dashboard interface"
                width={980}
                height={620}
                className="w-full h-auto"
                loading="eager"
                fetchPriority="high"
              />
            </div>
          </div>
        </div>
      </main>

      {/* FEATURES OVERVIEW SECTION */}
      <section id="features" className="pb-12 md:pb-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">Overview</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Features
            </h2>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              Postgresus provides everything you need for reliable PostgreSQL
              backup management. From automated scheduling to backups
              encryption. Suitable well for both individual developers with
              personal projects, DevOps teams and enterprises.
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          {/* Feature Cards Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 border border-[#ffffff20] rounded-xl">
            {/* Card 1: Scheduled backups */}
            <div className="border-b md:border-r lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                1
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Scheduled backups
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Scheduled backups"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Backup is a thing that should be done in specified time on
                regular basis. So we provide many options: from hourly to
                monthly
              </p>
            </div>

            {/* Card 2: Configurable health checks */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Configurable health checks
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="Health checks"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base mb-3">
                Each minute (or any another amount of time) the system will ping
                your database and show you the history of attempts
              </p>

              <p className="text-gray-400 text-sm md:text-base">
                The database can be considered as down after 3 failed attempts
                or so. Once DB is healthy again - you receive notification too
              </p>
            </div>

            {/* Card 3: Many destinations to store */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                3
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Many destinations to store
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Files are kept on VPS, cloud storages and other places. You can
                choose any storage you. Files are always owned by you.{" "}
                <a
                  href="/storages"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  View all →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-destinations.svg"
                  alt="Storage"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 4: Notifications */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                4
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Notifications
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                You can receive notifications about success or fail of the
                process. This is useful for developers or DevOps teams.{" "}
                <a
                  href="/notifiers"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  View all →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-notifications.svg"
                  alt="Notifications"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 5: Self hosted via Docker */}
            <div className="border-b md:border-r lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                5
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Self hosted via Docker
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                Postgresus runs on your PC or VPS. Therefore, all your data is
                owned by you and secured. Deploy takes about 2 minutes via
                script, Docker or k8s
              </p>

              <div className="flex">
                <img
                  src="/images/index/feature-deploy.svg"
                  alt="Docker"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 6: Open source and free */}
            <div className="border-b border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                6
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Open source and free
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                The project is fully open source, free and have Apache 2.0
                license. You can copy and fork the code on your own. Open for
                enterprise as well
              </p>
              <div>
                <img
                  src="/images/index/feature-github.svg"
                  alt="GitHub"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 7: Many PostgreSQL versions - Mobile/Tablet separate, Desktop merged with card 10 */}
            <div className="border-b md:border-r lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1 lg:row-span-2 lg:flex lg:flex-col">
              {/* Card 7: Many PostgreSQL versions */}
              <div className="lg:border-b lg:border-[#ffffff20] lg:pb-6 lg:mb-0">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  7
                </div>

                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Many PostgreSQL versions
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  PostgreSQL 13, 14, 15, 16, 17 and 18 are supported by the
                  project. You can backup any version from 2019
                </p>

                <div>
                  <img
                    src="/images/index/feature-postgresql.svg"
                    alt="PostgreSQL"
                    loading="lazy"
                  />
                </div>
              </div>

              {/* Card 10: Security - Only visible on desktop, merged with card 7 */}
              <div className="hidden lg:block lg:pt-6">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  10
                </div>

                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Security
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  Enterprise-grade encryption protects sensitive data and
                  backups. Read-only database access prevents data corruption.
                  Everything this does not require any knowledge and ready out
                  of the box from the start automatically.{" "}
                  <a
                    href="/security"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Read more →
                  </a>
                </p>

                <div>
                  <img
                    src="/images/index/feature-encryption.svg"
                    alt="Security"
                    loading="lazy"
                  />
                </div>
              </div>
            </div>

            {/* Card 8: Access management */}
            <div className="border-b border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold border border-[#ffffff20]">
                  8
                </div>
              </div>

              <div className="flex flex-wrap items-center mb-4 md:mb-5">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold">
                  Access management
                </h3>

                <div className="px-2 py-1 rounded border border-[#ffffff20] text-sm font-medium ml-2">
                  for teams
                </div>
              </div>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-access-management.svg"
                  alt="Access management"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Provide access for users to view or manage DBs. Separate teams
                and projects. Suitable for DevOps teams and developers.{" "}
                <a
                  href="/access-management#settings"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Read more →
                </a>
              </p>
            </div>

            {/* Card 9: Audit logs */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold border border-[#ffffff20]">
                  9
                </div>
              </div>

              <div className="flex flex-wrap items-center mb-4 md:mb-5">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold">
                  Audit logs
                </h3>

                <div className="px-2 py-1 rounded border border-[#ffffff20] text-sm font-medium ml-2">
                  for teams
                </div>
              </div>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-audit-logs.svg"
                  alt="Audit logs"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Track all system activities with comprehensive audit logs. You
                can view access and changes history for each user (backup
                downloads, schedule changes, config updates, etc.).{" "}
                <a
                  href="/access-management#audit-logs"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Read more →
                </a>
              </p>
            </div>

            {/* Card 10: Security - Mobile/Tablet only */}
            <div className="border-b border-[#ffffff20] p-5 md:p-6 col-span-1 lg:hidden">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                10
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Security
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                Enterprise-grade encryption protects sensitive data and backups.
                Read-only database access prevents data corruption. Everything
                this does not require any knowledge and ready out of the box
                from the start automatically.{" "}
                <a
                  href="/security"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Read more →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-encryption.svg"
                  alt="Security"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Try Postgresus CTA */}
            <div className="col-span-1 md:col-span-2 lg:col-span-2 p-5 md:p-6 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 bg-[#16181D]">
              <div>
                <div className="text-lg md:text-xl font-bold mb-2">
                  Launch Postgresus in 2 minutes
                </div>

                <p className="text-gray-400 text-sm md:text-base max-w-[350px]">
                  You&apos;ll only need about 2 minutes to configure Postgresus
                  and start managing backups
                </p>
              </div>

              <a
                href="#installation"
                className="w-full sm:w-auto inline-flex items-center justify-center gap-2 px-5 py-2.5 text-black bg-white rounded-lg font-medium hover:opacity-70 transition-colors shrink-0"
              >
                <span>Get started</span>
                <svg
                  aria-hidden={true}
                  width="18"
                  height="18"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <path d="M5 12h14M12 5l7 7-7 7" />
                </svg>
              </a>
            </div>
          </div>
        </div>
      </section>

      {/* VIDEO SECTION */}
      <section className="pb-12 md:pb-20 px-4 md:px-6 lg:px-0" id="how-to-use">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col lg:flex-row gap-8 lg:gap-16">
            {/* Left side: Info */}
            <div className="w-full lg:w-[450px] lg:shrink-0">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">4-minutes overview</span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                How to use Postgresus?
              </h2>

              <p className="text-gray-200 max-w-[450px] leading-relaxed mb-6 md:mb-8 text-sm sm:text-base">
                Watch in this video how to connect your database, how to
                configure backups schedule, how to download and restore backups,
                how to add team members and what is users&apos; audit logs
              </p>

              <a
                href="https://rostislav-dugin.com"
                target="_blank"
                className="flex items-center gap-3 md:gap-4 hover:opacity-70 transition-colors"
              >
                <img
                  src="/images/index/rostislav.png"
                  alt="Rostislav Dugin"
                  className="w-10 h-10 md:w-12 md:h-12 rounded-full object-cover"
                  loading="lazy"
                />

                <div>
                  <p className="font-medium text-base md:text-lg">
                    Rostislav Dugin
                  </p>
                  <p className="text-sm text-gray-400">
                    Developer of Postgresus
                  </p>
                </div>
              </a>
            </div>

            {/* Right side: Video */}
            <div className="flex-1 relative">
              <div className="rounded-lg overflow-hidden shadow-lg border border-[#ffffff20]">
                <LiteYouTubeEmbed
                  videoId="1qsAnijJfJE"
                  title="How to use Postgresus (overview)?"
                  thumbnailSrc="/images/index/how-to-use-preview.svg"
                />
              </div>
            </div>
          </div>
        </div>
      </section>

      <div className="border-b border-[#ffffff20] max-w-[calc(100%-2rem)] md:max-w-[calc(100%-3rem)] lg:max-w-[1000px] 2xl:max-w-[1200px] mx-auto" />

      {/* PROCESS SECTION */}
      <section className="py-12 md:py-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center mb-10 md:mb-16">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">Process</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              How to make backups?
            </h2>

            <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
              The main priority for Postgresus is simplicity, right now this is
              the easiest tool to backup PostgreSQL in the world. To make
              backups, you need to follow 4 steps. After that, you will be able
              to restore in one click
            </p>
          </div>

          {/* Steps */}
          <div className="space-y-6 md:space-y-10 max-w-[1000px] mx-auto">
            {/* Step 1 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Step 1
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Select required schedule
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-200 text-sm md:text-base">
                  <p>
                    You can choose any time you need: daily, weekly, monthly and
                    particular time (like 4 AM)
                  </p>
                  <p>
                    For week interval you need to specify particular day, for
                    month you need to specify particular day
                  </p>
                  <p>
                    If your database is large, we recommend you choosing the
                    time when there are decrease in traffic
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Step 1"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Step 2 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Step 2
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Enter your database info
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-200 text-sm md:text-base">
                  <p>
                    Enter credentials of your PostgreSQL database, select
                    version and target DB. Also choose whether SSL is required
                  </p>
                  <p>
                    Postgresus, by default, compress backups at balance level to
                    not slow down backup process (~20% slower) and save x4-x8 of
                    the space (that decreasing network traffic)
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-2.svg"
                  alt="Step 2"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Step 3 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Step 3
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Choose storage for backups
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-200 text-sm md:text-base">
                  <p>
                    You can keep files with backups locally, in S3, in Google
                    Drive, NAS, Dropbox and other services
                  </p>
                  <p>
                    Please keep in mind that you need to have enough space on
                    the storage
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-3.svg"
                  alt="Step 3"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Step 4 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Step 4
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Choose where you want to receive notifications (optional)
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-200 text-sm md:text-base">
                  <p>
                    When backup succeed or failed, Postgresus is able to send
                    you notification. It can be chat with DevOps, your emails or
                    even webhook of your team
                  </p>
                  <p>
                    We are going to support the most of popular messangers and
                    platforms
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-4.svg"
                  alt="Step 4"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>
          </div>

          {/* CTA Button */}
          <div className="text-center mt-8 md:mt-12">
            <a
              href="#installation"
              className="inline-flex items-center gap-2 px-6 py-3 bg-white text-black rounded-lg text-[15px] font-medium hover:opacity-70 transition-colors"
            >
              Get started
              <svg
                aria-hidden={true}
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M5 12h14M12 5l7 7-7 7" />
              </svg>
            </a>
          </div>
        </div>
      </section>

      {/* INSTALLATION SECTION */}
      <section id="installation" className="px-4 md:px-6 lg:px-0">
        <div className="max-w-[1000px] 2xl:max-w-[1200px] mx-auto border border-[#ffffff20] rounded-xl py-10 md:py-20 px-4 md:px-6">
          <div className="max-w-[1100px] mx-auto">
            <div className="text-center mb-8 md:mb-10">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">Get started</span>
              </div>

              <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
                How to install?
              </h2>

              <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
                Postgresus support many ways of installation. Both local and
                cloud are supported. Both ways are extremely simple and easy to
                use even for those who has no experience in administration or
                DevOps
              </p>
            </div>

            <InstallationComponent />
          </div>
        </div>
      </section>

      {/* FAQ SECTION */}
      <section id="faq" className="py-12 md:py-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center mb-8 md:mb-12">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">FAQ</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Frequent questions
            </h2>

            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              The goal of Postgresus — make backing up as simple as possible for
              single developers (as well as DevOps) and teams. UI makes it easy
              to create backups and visualizes the progress and restores
              anything in couple of clicks
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="What is Postgresus and why should I use it instead of hand-rolled scripts?"
              answer="Postgresus is an Apache 2.0 licensed, self-hosted service backing up PostgreSQL, v12 to v18. It differs from shell scripts in that it has a frontend for scheduling tasks, compressing and storing archives on multiple targets (local disk, S3, Google Drive, NAS, Dropbox, etc.) and notifying your team when tasks finish or fail — all without hand-rolled code"
            />
            <FaqItem
              number="3"
              question="What backup schedules can I schedule?"
              answer="You can choose from hourly, daily, weekly or monthly cycles and even choose an exact run time (such as 04:00 when it's late night). Weekly schedules enable you to choose a particular weekday, while monthly schedules enable you to choose a particular calendar day, giving you very fine-grained control of maintenance windows."
            />
            <FaqItem
              number="5"
              question="How will I know a backup succeeded — or worse, failed?"
              answer="Postgresus can notify with real-time emails, Slack, Telegram, webhooks, Mattermost, Discord and more. You have the choice of what channels to ping so that your DevOps team hears about successes and failures in real time, making recovery routines and compliance audits easier."
            />
            <FaqItem
              number="7"
              question="How do I set up and run my first backup job in Postgresus?"
              answer={
                <>
                  To start your very first Postgresus backup, simply log in to
                  the dashboard, click on New Backup, select an interval —
                  hourly, daily, weekly or monthly. Then specify the exact run
                  time (e.g., 02:30 for off-peak hours).
                  <br />
                  <br />
                  Then input your PostgreSQL host, port number, database name,
                  credentials, version and SSL preference. Choose where the
                  archive should be sent (local path, S3 bucket, Google Drive
                  folder, Dropbox, etc.). <br />
                  <br />
                  If you need, add notification channels such as email, Slack,
                  Telegram or a webhook and click Save. Postgresus instantly
                  validates the info, starts the schedule, runs the initial job
                  and sends live status. So you may restore with one touch when
                  the backup is complete.
                </>
              }
            />
            <FaqItem
              number="9"
              question="Who is Postgresus suitable for?"
              answer="Postgresus is designed for single developers, DevOps teams, organizations, startups, system administrators and IT departments who need reliable PostgreSQL backups. Whether you're managing personal projects or production databases, Postgresus provides enterprise-grade backup capabilities with a simple, intuitive interface."
            />
            <FaqItem
              number="11"
              question="Can I use Postgresus as an individual and as a team?"
              answer={
                <>
                  Yes, Postgresus works perfectly for both individual developers
                  and teams. For individuals, you can manage all your databases
                  with a simple, secure interface. For teams, Postgresus offers
                  access management features that let you create multiple users
                  with different permission levels (viewer, editor, admin).
                  <br />
                  <br />
                  You can control who can view or manage specific databases,
                  making it ideal for DevOps teams and development
                  organizations. Additionally, audit logs track all system
                  activities, showing who accessed what and when, which is
                  essential for security compliance and team accountability.
                </>
              }
            />
            <FaqItem
              number="13"
              question="Is Postgresus an alternative to pg_dump?"
              answer="Yes, Postgresus is a modern alternative to pg_dump. Under the hood, Postgresus uses pg_dump for creating backups, but extends it with a user-friendly web interface, automated scheduling, multiple storage destinations, real-time notifications, health monitoring and backup encryption. Think of Postgresus as pg_dump with superpowers — you get all the reliability of pg_dump plus enterprise features without writing shell scripts."
            />

            <FaqItem
              number="2"
              question="How do I install Postgresus in the quickest manner?"
              answer="Postgresus supports multiple installation methods: automated script, Docker, Docker Compose and Kubernetes with Helm. The quickest route is to run the one-line cURL installer, which fetches the current Docker image, creates a docker-compose.yml and boots up the service so it will automatically restart on reboots. For Kubernetes environments, use the official Helm chart for production-ready deployments. Overall time is usually less than two minutes on a typical VPS."
            />
            <FaqItem
              number="4"
              question="Where do my backups live and how much space will they occupy?"
              answer="Archives can be saved to local volumes, S3-compatible buckets, Google Drive, Dropbox and other cloud targets. Postgresus implements balanced compression, which typically shrinks dump size by 4-8x with incremental only about 20% of runtime overhead, so you have storage and bandwidth savings."
            />
            <FaqItem
              number="6"
              question="How does Postgresus ensure security?"
              answer="Postgresus enforces security on three levels: (1) Sensitive data encryption — all passwords, tokens and credentials are encrypted with AES-256-GCM and stored separately from the database; (2) Backup encryption — each backup file is encrypted with a unique key derived from a master key, backup ID and random salt, making backups useless without your encryption key even if someone gains storage access; (3) Read-only database access — Postgresus only requires SELECT permissions and performs comprehensive checks to ensure no write privileges exist, preventing data corruption even if the tool is compromised."
            />
            <FaqItem
              number="8"
              question="How does PostgreSQL monitoring work?"
              answer="Postgresus monitors your databases instantly. This optional feature helps avoid extra costs for edge DBs. Health checks are performed once a specific period (minute, 5 minutes, etc.). To enable the feature, choose your DB and select 'enable' monitoring. Then configure health checks period and number of failed attempts to consider the DB as unavailable."
            />
            <FaqItem
              number="10"
              question="How is Postgresus different from PgBackRest, Barman or pg_dump?"
              answer="Postgresus provides a modern, user-friendly web interface instead of complex configuration files and command-line tools. While PgBackRest and Barman require extensive configuration and command-line expertise, Postgresus offers intuitive point-and-click setup. Unlike raw pg_dump scripts, it includes built-in scheduling, compression, multiple storage destinations, health monitoring and real-time notifications — all managed through a simple web UI."
            />
            <FaqItem
              number="12"
              question="Where can I read comparisons with other PostgreSQL backup tools?"
              answer={
                <>
                  We have detailed comparison pages for popular backup tools:{" "}
                  <a
                    href="/pgdump-alternative"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Postgresus vs pg_dump
                  </a>
                  ,{" "}
                  <a
                    href="/postgresus-vs-pgbackrest"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Postgresus vs pgBackRest
                  </a>
                  ,{" "}
                  <a
                    href="/postgresus-vs-barman"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Postgresus vs Barman
                  </a>
                  ,{" "}
                  <a
                    href="/postgresus-vs-wal-g"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Postgresus vs WAL-G
                  </a>{" "}
                  and{" "}
                  <a
                    href="/postgresus-vs-pgbackweb"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Postgresus vs pgBackWeb
                  </a>
                  . Each comparison explains the key differences, pros and cons,
                  and helps you choose the right tool for your needs.
                </>
              }
            />
          </div>
        </div>
      </section>

      {/* FOOTER */}
      <footer className="py-8 md:py-12 border-t border-[#ffffff20] px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col items-center">
            <Link href="/" className="flex items-center gap-2.5 mb-6">
              <img
                src="/logo.svg"
                alt="Postgresus logo"
                width={32}
                height={32}
                className="h-7 w-7 md:h-8 md:w-8"
              />

              <span className="text-base md:text-lg font-semibold">
                Postgresus
              </span>
            </Link>

            <div className="flex flex-wrap items-center justify-center gap-4 md:gap-6 mb-4 text-sm md:text-base">
              <a
                href="/installation"
                className="hover:text-gray-200 transition-colors"
              >
                Documentation
              </a>
              <a
                href="https://github.com/RostislavDugin/postgresus"
                target="_blank"
                rel="noopener noreferrer"
                className="hover:text-gray-200 transition-colors"
              >
                GitHub
              </a>
              <a
                href="https://t.me/postgresus_community"
                target="_blank"
                rel="noopener noreferrer"
                className="hover:text-gray-200 transition-colors"
              >
                Community
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

            <p className="text-gray-400 text-sm md:text-base text-center">
              © 2025 Postgresus. All rights reserved.
            </p>
          </div>
        </div>
      </footer>
    </div>
  );
}

function FaqItem({
  number,
  question,
  answer,
}: {
  number: string;
  question: string;
  answer: React.ReactNode;
}) {
  return (
    <div className="rounded-lg border border-[#ffffff20] p-4 md:p-6">
      <div className="flex items-center justify-center w-6 h-6 rounded border border-[#ffffff20] text-sm font-semibold mb-3 md:mb-4">
        {number}
      </div>

      <h3 className="text-base md:text-lg font-bold mb-2 md:mb-3">
        {question}
      </h3>

      <div className="text-gray-400 text-sm md:text-base">{answer}</div>
    </div>
  );
}
