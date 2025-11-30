import type { Metadata } from "next";
import DocsNavbarComponent from "../components/DocsNavbarComponent";
import DocsSidebarComponent from "../components/DocsSidebarComponent";
import DocTableOfContentComponent from "../components/DocTableOfContentComponent";
import { CopyButton } from "../components/CopyButton";

export const metadata: Metadata = {
  title: "FAQ - Frequently Asked Questions | Postgresus",
  description:
    "Frequently asked questions about Postgresus PostgreSQL backup tool. Learn how to backup localhost databases, understand backup formats, compression methods and more.",
  keywords: [
    "Postgresus FAQ",
    "PostgreSQL backup questions",
    "localhost database backup",
    "backup formats",
    "pg_dump compression",
    "zstd compression",
    "PostgreSQL backup help",
    "database backup guide",
  ],
  openGraph: {
    title: "FAQ - Frequently Asked Questions | Postgresus",
    description:
      "Frequently asked questions about Postgresus PostgreSQL backup tool. Learn how to backup localhost databases, understand backup formats, compression methods and more.",
    type: "article",
    url: "https://postgresus.com/faq",
  },
  twitter: {
    card: "summary",
    title: "FAQ - Frequently Asked Questions | Postgresus",
    description:
      "Frequently asked questions about Postgresus PostgreSQL backup tool. Learn how to backup localhost databases, understand backup formats, compression methods and more.",
  },
  alternates: {
    canonical: "https://postgresus.com/faq",
  },
  robots: "index, follow",
};

export default function FAQPage() {
  const dockerComposeHost = `services:
  postgresus:
    container_name: postgresus
    image: rostislavdugin/postgresus:latest
    network_mode: host
    volumes:
      - ./postgresus-data:/postgresus-data
    restart: unless-stopped`;

  const dockerRunHost = `docker run -d \\
  --name postgresus \\
  --network host \\
  -v ./postgresus-data:/postgresus-data \\
  --restart unless-stopped \\
  rostislavdugin/postgresus:latest`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "FAQPage",
            mainEntity: [
              {
                "@type": "Question",
                name: "How to backup localhost databases?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "To backup databases running on localhost, you need to configure Postgresus to use host network mode in Docker. This allows the container to access services running on your host machine (localhost).",
                },
              },
              {
                "@type": "Question",
                name: "Why does Postgresus not use raw SQL dump format?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Postgresus uses the directory format with zstd compression because it provides the most efficient backup and restore speed after extensive testing. The directory format with zstd compression level 5 offers the optimal balance between backup creation speed, restore speed and file size.",
                },
              },
              {
                "@type": "Question",
                name: "Where is Postgresus installed?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Postgresus is installed in /opt/postgresus/",
                },
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
              <h1 id="faq">Frequently Asked Questions</h1>

              <p className="text-lg text-gray-700">
                Find answers to the most common questions about Postgresus,
                including installation, configuration and backup strategies.
              </p>

              <h2 id="how-to-backup-localhost">
                How to backup localhost databases?
              </h2>

              <p>
                If you&apos;re running Postgresus in Docker and want to back up
                databases running on your host machine (localhost), you need to
                configure Docker to use <strong>host network mode</strong>.
              </p>

              <p>
                By default, Docker containers run in an isolated network and
                cannot access services on <code>localhost</code>. The host
                network mode allows the container to share the host&apos;s
                network namespace.
              </p>

              <h3 id="docker-compose-solution">Solution for Docker Compose:</h3>

              <p>
                Update your <code>docker-compose.yml</code> file to use{" "}
                <code>network_mode: host</code>:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerComposeHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerComposeHost} />
                </div>
              </div>

              <h3 id="docker-run-solution">Solution for Docker run:</h3>

              <p>
                Use the <code>--network host</code> flag:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerRunHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerRunHost} />
                </div>
              </div>

              <div className="rounded-lg border border-blue-200 bg-blue-50 p-4 my-6 pb-0">
                <p className="text-sm text-blue-900 m-0">
                  <strong>💡 Note:</strong> When using host network mode, you
                  can connect to your localhost database using{" "}
                  <code>127.0.0.1</code> or <code>localhost</code> as the host
                  in your Postgresus backup configuration. You&apos;ll also
                  access the Postgresus UI directly at{" "}
                  <code>http://localhost:4005</code> without port mapping.
                </p>
              </div>

              <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 my-6 pb-0">
                <p className="text-sm text-amber-900 m-0">
                  <strong>⚠️ Important for Windows and macOS users:</strong> The{" "}
                  <code>host</code> network mode only works natively on Linux.
                  On Windows and macOS, Docker runs inside a Linux VM, so{" "}
                  <code>host.docker.internal</code> should be used instead of{" "}
                  <code>localhost</code> as the database host address in your
                  backup configuration.
                </p>
              </div>

              <h2 id="why-no-raw-sql-dump">
                Why does Postgresus not use raw SQL dump format?
              </h2>

              <p>
                Postgresus uses the <code>pg_dump</code>&apos;s{" "}
                <strong>directory format</strong> with{" "}
                <strong>zstd compression at level 5</strong> instead of the
                plain SQL format because it provides the most efficient balance
                between:
              </p>

              <ul>
                <li>Backup creation speed</li>
                <li>Restore speed</li>
                <li>
                  File size compression (up to 20x times smaller than plain SQL
                  format)
                </li>
              </ul>

              <p>
                This decision was made after extensive testing and benchmarking
                of different PostgreSQL backup formats and compression methods.
                You can read more about testing here{" "}
                <a
                  href="https://dev.to/rostislav_dugin/postgresql-backups-comparing-pgdump-speed-in-different-formats-and-with-different-compression-4pmd"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  PostgreSQL backups: comparing pg_dump speed in different
                  formats and with different compression
                </a>
                .
              </p>

              <p>Postgresus will not include raw SQL dump format, because:</p>

              <ul>
                <li>extra variety is bad for UX;</li>
                <li>makes it harder to support the code;</li>
                <li>current dump format is suitable for 99% of the cases</li>
              </ul>

              <p>
                As for Nov 2025 Postgresus is going to add incremental backups
                support. So this is priority in addition to current dump format.
              </p>

              <h2 id="installation-directory">
                Where is Postgresus installed if installed via .sh script?
              </h2>

              <p>
                Postgresus is installed in <code>/opt/postgresus/</code>{" "}
                directory.
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
