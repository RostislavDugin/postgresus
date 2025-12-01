import type { Metadata } from "next";
import Link from "next/link";
import DocsNavbarComponent from "../components/DocsNavbarComponent";
import DocsSidebarComponent from "../components/DocsSidebarComponent";
import DocTableOfContentComponent from "../components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "pg_dump Alternative - Postgresus PostgreSQL Backup Tool",
  description:
    "Postgresus is built on pg_dump and extends its features with a web UI, automated scheduling, cloud storage, notifications, team collaborationand encryption.",
  keywords: [
    "pg_dump alternative",
    "pg_dump GUI",
    "pg_dump automation",
    "pg_dump web interface",
    "PostgreSQL backup tool",
    "pg_dump scheduler",
    "pg_dump cloud storage",
    "pg_dump encryption",
    "PostgreSQL backup automation",
    "pg_dump wrapper",
  ],
  openGraph: {
    title: "pg_dump Alternative - Postgresus PostgreSQL Backup Tool",
    description:
      "Postgresus is built on pg_dump and extends its features with a web UI, automated scheduling, cloud storage, notifications, team collaborationand encryption.",
    type: "article",
    url: "https://postgresus.com/pgdump-alternative",
  },
  twitter: {
    card: "summary",
    title: "pg_dump Alternative - Postgresus PostgreSQL Backup Tool",
    description:
      "Postgresus is built on pg_dump and extends its features with a web UI, automated scheduling, cloud storage, notifications, team collaborationand encryption.",
  },
  alternates: {
    canonical: "https://postgresus.com/pgdump-alternative",
  },
  robots: "index, follow",
};

export default function PgDumpAlternativePage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "pg_dump Alternative - Postgresus PostgreSQL Backup Tool",
            description:
              "A comprehensive guide to Postgresus as a pg_dump alternative, explaining how it builds on pg_dump and extends its capabilities with automation, cloud storage, notificationsand team features.",
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

      <div className="flex min-h-screen">
        {/* Sidebar */}
        <DocsSidebarComponent />

        {/* Main Content */}
        <main className="flex-1 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="pgdump-alternative">pg_dump Alternative</h1>

              <p className="text-lg text-gray-700">
                Postgresus is a PostgreSQL backup tool built on top of{" "}
                <code>pg_dump</code>. Rather than replacing <code>pg_dump</code>
                , Postgresus extends its capabilities with a web interface,
                automated scheduling, cloud storage integration, notifications,
                team collaboration featuresand built-in encryption.
              </p>

              <h2 id="quick-comparison">Quick comparison</h2>

              <p>
                Here&apos;s an overview of how Postgresus extends the core{" "}
                <code>pg_dump</code> functionality:
              </p>

              <div className="not-prose overflow-x-auto my-6">
                <table className="w-full border-collapse rounded-lg overflow-hidden shadow-sm border border-gray-200">
                  <thead>
                    <tr className="bg-gray-100 border-b border-gray-200">
                      <th className="px-4 py-3 text-center font-semibold text-gray-900">
                        Feature
                      </th>
                      <th className="px-4 py-3 text-center font-semibold text-gray-900">
                        pg_dump
                      </th>
                      <th className="px-4 py-3 text-center font-semibold text-gray-900">
                        Postgresus
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200">
                    <tr className="bg-white hover:bg-gray-50 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Backup engine
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        pg_dump
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        <span className="inline-flex items-center rounded-full bg-blue-100 px-2.5 py-0.5 text-xs font-medium text-blue-800">
                          Built on pg_dump
                        </span>
                      </td>
                    </tr>
                    <tr className="bg-gray-50 hover:bg-gray-100 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Interface
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Command-line
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Web UI + API
                      </td>
                    </tr>
                    <tr className="bg-white hover:bg-gray-50 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Scheduling
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Manual or cron scripts
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        <span className="inline-flex items-center gap-1">
                          <svg
                            className="w-4 h-4 text-green-500"
                            fill="currentColor"
                            viewBox="0 0 20 20"
                          >
                            <path
                              fillRule="evenodd"
                              d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                              clipRule="evenodd"
                            />
                          </svg>
                          Built-in scheduler
                        </span>
                      </td>
                    </tr>
                    <tr className="bg-gray-50 hover:bg-gray-100 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Storage destinations
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Local filesystem
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Local, S3, Google Drive, R2, Azure, NAS, Dropbox
                      </td>
                    </tr>
                    <tr className="bg-white hover:bg-gray-50 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Compression
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        gzip, LZ4, zstd (manual)
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        <span className="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800">
                          zstd (automatic, optimized)
                        </span>
                      </td>
                    </tr>
                    <tr className="bg-gray-50 hover:bg-gray-100 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Encryption
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        External tools required
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        <span className="inline-flex items-center gap-1">
                          <svg
                            className="w-4 h-4 text-green-500"
                            fill="currentColor"
                            viewBox="0 0 20 20"
                          >
                            <path
                              fillRule="evenodd"
                              d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                              clipRule="evenodd"
                            />
                          </svg>
                          AES-256-GCM built-in
                        </span>
                      </td>
                    </tr>
                    <tr className="bg-white hover:bg-gray-50 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Notifications
                      </td>
                      <td className="px-4 py-3 text-center text-gray-500">
                        <span className="inline-flex items-center gap-1">
                          <svg
                            className="w-4 h-4 text-red-400"
                            fill="currentColor"
                            viewBox="0 0 20 20"
                          >
                            <path
                              fillRule="evenodd"
                              d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                              clipRule="evenodd"
                            />
                          </svg>
                          None
                        </span>
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        <span className="inline-flex items-center gap-1">
                          <svg
                            className="w-4 h-4 text-green-500"
                            fill="currentColor"
                            viewBox="0 0 20 20"
                          >
                            <path
                              fillRule="evenodd"
                              d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                              clipRule="evenodd"
                            />
                          </svg>
                          Slack, Teams, Telegram, Email, Webhooks
                        </span>
                      </td>
                    </tr>
                    <tr className="bg-gray-50 hover:bg-gray-100 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Team features
                      </td>
                      <td className="px-4 py-3 text-center text-gray-500">
                        <span className="inline-flex items-center gap-1">
                          <svg
                            className="w-4 h-4 text-red-400"
                            fill="currentColor"
                            viewBox="0 0 20 20"
                          >
                            <path
                              fillRule="evenodd"
                              d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                              clipRule="evenodd"
                            />
                          </svg>
                          None
                        </span>
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        <span className="inline-flex items-center gap-1">
                          <svg
                            className="w-4 h-4 text-green-500"
                            fill="currentColor"
                            viewBox="0 0 20 20"
                          >
                            <path
                              fillRule="evenodd"
                              d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                              clipRule="evenodd"
                            />
                          </svg>
                          Workspaces, RBAC, audit logs
                        </span>
                      </td>
                    </tr>
                    <tr className="bg-white hover:bg-gray-50 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Retention policies
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Manual cleanup scripts
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        <span className="inline-flex items-center gap-1">
                          <svg
                            className="w-4 h-4 text-green-500"
                            fill="currentColor"
                            viewBox="0 0 20 20"
                          >
                            <path
                              fillRule="evenodd"
                              d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                              clipRule="evenodd"
                            />
                          </svg>
                          Automatic retention
                        </span>
                      </td>
                    </tr>
                    <tr className="bg-gray-50 hover:bg-gray-100 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Health monitoring
                      </td>
                      <td className="px-4 py-3 text-center text-gray-500">
                        <span className="inline-flex items-center gap-1">
                          <svg
                            className="w-4 h-4 text-red-400"
                            fill="currentColor"
                            viewBox="0 0 20 20"
                          >
                            <path
                              fillRule="evenodd"
                              d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                              clipRule="evenodd"
                            />
                          </svg>
                          None
                        </span>
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        <span className="inline-flex items-center gap-1">
                          <svg
                            className="w-4 h-4 text-green-500"
                            fill="currentColor"
                            viewBox="0 0 20 20"
                          >
                            <path
                              fillRule="evenodd"
                              d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                              clipRule="evenodd"
                            />
                          </svg>
                          Built-in health checks
                        </span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>

              <h2 id="what-is-pgdump">What is pg_dump?</h2>

              <p>
                <code>pg_dump</code> is PostgreSQL&apos;s native utility for
                creating logical backups. It&apos;s been part of PostgreSQL
                since the beginning and is the standard tool for database
                exports.
              </p>

              <h3 id="pgdump-strengths">pg_dump strengths</h3>

              <ul>
                <li>
                  <strong>Portable backups</strong>: Creates SQL or custom
                  format dumps that can be restored to different PostgreSQL
                  versions.
                </li>
                <li>
                  <strong>Selective backups</strong>: Can export specific
                  tables, schemasor entire databases.
                </li>
                <li>
                  <strong>Consistent snapshots</strong>: Uses PostgreSQL&apos;s
                  MVCC to create consistent backups without blocking writes.
                </li>
                <li>
                  <strong>Widely supported</strong>: Available on every
                  PostgreSQL installation, well-documentedand battle-tested.
                </li>
                <li>
                  <strong>Flexible output formats</strong>: Plain SQL, custom,
                  directoryor tar formats.
                </li>
              </ul>

              <h3 id="pgdump-limitations">pg_dump limitations</h3>

              <p>
                While <code>pg_dump</code> is powerful, using it in production
                typically requires additional scripting:
              </p>

              <ul>
                <li>
                  <strong>No built-in scheduling</strong>: Requires cron jobs or
                  external schedulers.
                </li>
                <li>
                  <strong>Local storage only</strong>: Outputs to local
                  filesystem; cloud uploads require additional scripts.
                </li>
                <li>
                  <strong>No encryption</strong>: Backup files are unencrypted
                  by default; requires piping through gpg or similar tools.
                </li>
                <li>
                  <strong>No notifications</strong>: No way to alert on backup
                  success or failure without custom scripting.
                </li>
                <li>
                  <strong>No retention management</strong>: Old backups must be
                  cleaned up manually or via scripts.
                </li>
                <li>
                  <strong>Command-line only</strong>: No visual interface for
                  monitoring or management.
                </li>
              </ul>

              <h2 id="how-postgresus-extends">
                How Postgresus extends pg_dump
              </h2>

              <p>
                Postgresus uses <code>pg_dump</code> as its backup engine,
                preserving all the benefits of logical backups while adding
                enterprise features on top.
              </p>

              <div className="rounded-lg border border-blue-200 bg-blue-50 p-4 my-6">
                <p className="text-blue-900 m-0">
                  <strong>Under the hood:</strong> When you trigger a backup in
                  Postgresus, it executes <code>pg_dump</code> with optimized
                  parameters, then handles compression, encryptionand upload to
                  your configured storage destination.
                </p>
              </div>

              <h3 id="web-interface">Web interface</h3>

              <p>
                Instead of remembering <code>pg_dump</code> command-line
                options, Postgresus provides a web UI where you can:
              </p>

              <ul>
                <li>Add databases with a guided connection wizard</li>
                <li>Configure backup schedules with visual controls</li>
                <li>Monitor backup history and status at a glance</li>
                <li>Download or restore backups with one click</li>
                <li>View database health and availability charts</li>
              </ul>

              <h3 id="optimized-compression">Optimized compression</h3>

              <p>
                Postgresus uses zstd compression (level 5) by default, which
                provides:
              </p>

              <ul>
                <li>
                  <strong>4-8x size reduction</strong> compared to uncompressed
                  dumps
                </li>
                <li>
                  <strong>~20% runtime overhead</strong> — much faster than gzip
                </li>
                <li>
                  <strong>Automatic handling</strong> — no need to pipe through
                  compression tools
                </li>
              </ul>

              <h2 id="backup-automation">Backup automation</h2>

              <p>
                One of the most common challenges with <code>pg_dump</code> is
                setting up reliable automated backups.
              </p>

              <h3 id="automation-pgdump">Traditional pg_dump automation</h3>

              <p>
                A typical <code>pg_dump</code> automation script might look
                like:
              </p>

              <pre className="bg-gray-900 text-gray-100 rounded-lg p-4 overflow-x-auto">
                <code>{`#!/bin/bash
# Backup script for pg_dump
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups"
DB_NAME="mydb"

# Create backup
pg_dump -Fc -h localhost -U postgres $DB_NAME > $BACKUP_DIR/$DB_NAME_$DATE.dump

# Compress (if not using custom format)
# gzip $BACKUP_DIR/$DB_NAME_$DATE.sql

# Encrypt
gpg --encrypt --recipient backup@company.com $BACKUP_DIR/$DB_NAME_$DATE.dump

# Upload to S3
aws s3 cp $BACKUP_DIR/$DB_NAME_$DATE.dump.gpg s3://my-bucket/backups/

# Cleanup old backups (keep last 7 days)
find $BACKUP_DIR -name "*.dump*" -mtime +7 -delete

# Send notification on failure
if [ $? -ne 0 ]; then
  curl -X POST https://hooks.slack.com/... -d '{"text":"Backup failed!"}'
fi`}</code>
              </pre>

              <p>
                This script needs to be maintained, testedand monitored. Each
                database requires its own cron entry.
              </p>

              <h3 id="automation-postgresus">Postgresus automation</h3>

              <p>With Postgresus, the same functionality is built-in:</p>

              <ul>
                <li>
                  <strong>Visual scheduler</strong>: Set hourly, daily, weekly,
                  or monthly backups with specific times.
                </li>
                <li>
                  <strong>Automatic compression</strong>: zstd compression
                  applied automatically.
                </li>
                <li>
                  <strong>Built-in encryption</strong>: AES-256-GCM encryption
                  with unique keys per backup.
                </li>
                <li>
                  <strong>Cloud upload</strong>: Direct upload to S3, Google
                  Drive, Cloudflare R2, Azureor other destinations.
                </li>
                <li>
                  <strong>Retention policies</strong>: Automatic cleanup of old
                  backups based on your retention settings.
                </li>
                <li>
                  <strong>Notifications</strong>: Alerts to Slack, Teams,
                  Telegram, Email on success or failure.
                </li>
              </ul>

              <h2 id="storage-options">Storage options</h2>

              <p>
                <code>pg_dump</code> writes to the local filesystem. Getting
                backups to cloud storage requires additional tools and scripts.
              </p>

              <h3 id="storage-postgresus">Postgresus storage destinations</h3>

              <p>
                Postgresus supports multiple storage destinations out of the
                box:
              </p>

              <ul>
                <li>Local storage</li>
                <li>Amazon S3 and S3-compatible services</li>
                <li>Google Drive</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (Network-attached storage)</li>
                <li>Dropbox</li>
              </ul>

              <p>
                Each database can have its own storage destinationand you can
                configure multiple destinations for redundancy.
              </p>

              <p>
                <Link
                  href="/storages"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  View all storage options →
                </Link>
              </p>

              <h2 id="notifications">Notifications</h2>

              <p>
                Knowing when backups succeed or fail is critical for data
                protection.
              </p>

              <h3 id="notifications-pgdump">pg_dump notifications</h3>

              <p>
                <code>pg_dump</code> has no notification system. You need to:
              </p>

              <ul>
                <li>Write wrapper scripts that check exit codes</li>
                <li>Integrate with external monitoring tools</li>
                <li>Set up custom alerting pipelines</li>
              </ul>

              <h3 id="notifications-postgresus">Postgresus notifications</h3>

              <p>Postgresus includes built-in notifications to:</p>

              <ul>
                <li>Slack</li>
                <li>Discord</li>
                <li>Telegram</li>
                <li>Microsoft Teams</li>
                <li>Email</li>
                <li>Webhooks (for custom integrations)</li>
              </ul>

              <p>
                Configure which events trigger notifications: backup success,
                backup failureor both.
              </p>

              <p>
                <Link
                  href="/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  View all notification channels →
                </Link>
              </p>

              <h2 id="team-features">Team features</h2>

              <p>
                <code>pg_dump</code> is a single-user command-line tool.
                Postgresus adds collaboration features for teams:
              </p>

              <h3 id="team-postgresus">Postgresus team capabilities</h3>

              <ul>
                <li>
                  <strong>Workspaces</strong>: Organize databases, notifiers,
                  and storages by project or team. Users only see workspaces
                  they&apos;re invited to.
                </li>
                <li>
                  <strong>Role-based access control</strong>: Assign viewer,
                  editoror admin permissions to control what each team member
                  can do.
                </li>
                <li>
                  <strong>Audit logs</strong>: Track all system activities and
                  changes. Essential for security compliance and accountability.
                </li>
                <li>
                  <strong>Shared notifications</strong>: Team channels receive
                  backup status updates automatically.
                </li>
              </ul>

              <p>
                <Link
                  href="/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Learn more about access management →
                </Link>
              </p>

              <h2 id="security">Security</h2>

              <p>
                Security is where Postgresus adds significant value over raw{" "}
                <code>pg_dump</code> usage.
              </p>

              <h3 id="security-pgdump">pg_dump security</h3>

              <p>
                <code>pg_dump</code> creates unencrypted backup files. Securing
                them requires:
              </p>

              <ul>
                <li>Piping output through encryption tools (gpg, openssl)</li>
                <li>Managing encryption keys separately</li>
                <li>Ensuring secure key storage and rotation</li>
                <li>Setting up proper file permissions</li>
              </ul>

              <h3 id="security-postgresus">Postgresus security</h3>

              <p>Postgresus implements security at multiple levels:</p>

              <ul>
                <li>
                  <strong>AES-256-GCM encryption</strong>: All passwords, tokens
                  and credentials are encrypted. The encryption key is stored
                  separately from the database.
                </li>
                <li>
                  <strong>Unique backup encryption</strong>: Each backup file is
                  encrypted with a unique key derived from master key, backup
                  IDand random salt.
                </li>
                <li>
                  <strong>Read-only database access</strong>: Enforces SELECT
                  permissions only, preventing data corruption even if
                  compromised.
                </li>
              </ul>

              <p>
                <Link
                  href="/security"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Learn more about Postgresus security →
                </Link>
              </p>

              <h2 id="restore-process">Restore process</h2>

              <p>
                Both tools support restoring backups, but with different
                workflows.
              </p>

              <h3 id="restore-pgdump">Restoring pg_dump backups</h3>

              <p>
                Restoring a <code>pg_dump</code> backup requires:
              </p>

              <ol>
                <li>Locating the backup file</li>
                <li>Decrypting if encrypted</li>
                <li>Decompressing if compressed</li>
                <li>
                  Running <code>pg_restore</code> or <code>psql</code> with
                  correct parameters
                </li>
              </ol>

              <h3 id="restore-postgresus">Restoring Postgresus backups</h3>

              <p>Postgresus simplifies restoration:</p>

              <ul>
                <li>
                  <strong>One-click download</strong>: Download any backup
                  directly from the web interface.
                </li>
                <li>
                  <strong>Automatic decryption</strong>: Backups are decrypted
                  automatically when downloaded.
                </li>
                <li>
                  <strong>Restore commands provided</strong>: Postgresus shows
                  the exact <code>pg_restore</code> command for each backup.
                </li>
                <li>
                  <strong>Parallel restore support</strong>: Utilize multiple
                  CPU cores for faster restoration of large databases.
                </li>
              </ul>

              <h2 id="installation">Installation</h2>

              <h3 id="install-pgdump">pg_dump installation</h3>

              <p>
                <code>pg_dump</code> comes with PostgreSQL. If you have
                PostgreSQL installed, you have <code>pg_dump</code>.
              </p>

              <h3 id="install-postgresus">Postgresus installation</h3>

              <p>Postgresus offers multiple installation methods:</p>

              <ul>
                <li>
                  <strong>One-line script</strong>: Installs Docker (if needed),
                  sets up Postgresusand configures automatic startup.
                </li>
                <li>
                  <strong>Docker run</strong>: Single command to start with
                  embedded PostgreSQL.
                </li>
                <li>
                  <strong>Docker Compose</strong>: For more control over
                  deployment.
                </li>
              </ul>

              <p>
                <Link
                  href="/installation"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  View installation guide →
                </Link>
              </p>

              <h2 id="conclusion">Conclusion</h2>

              <p>
                <code>pg_dump</code> is PostgreSQL&apos;s proven backup utility,
                and Postgresus builds directly on top of it. The choice between
                using <code>pg_dump</code> directly or through Postgresus
                depends on your needs.
              </p>

              <div className="rounded-lg border border-gray-200 bg-gray-50 p-4 my-6">
                <p className="text-gray-900 m-0">
                  <strong>Use pg_dump directly if:</strong>
                </p>
                <ul className="text-gray-900 mb-0">
                  <li>You need one-off or ad-hoc database exports</li>
                  <li>
                    You&apos;re comfortable writing and maintaining shell
                    scripts
                  </li>
                  <li>
                    You have existing automation infrastructure (Ansible,
                    Terraform, etc.)
                  </li>
                  <li>You only need local backups without cloud storage</li>
                  <li>You&apos;re a single developer with simple needs</li>
                </ul>
              </div>

              <div className="rounded-lg border border-blue-200 bg-blue-50 p-4 my-6">
                <p className="text-blue-900 m-0">
                  <strong>Use Postgresus if:</strong>
                </p>
                <ul className="text-blue-900 mb-0">
                  <li>
                    You want automated, scheduled backups without writing
                    scripts
                  </li>
                  <li>
                    You need to store backups in cloud storage (S3, Google
                    Drive, etc.)
                  </li>
                  <li>
                    You want built-in encryption without managing keys manually
                  </li>
                  <li>You need notifications when backups succeed or fail</li>
                  <li>
                    You&apos;re working in a team and need collaboration
                    features
                  </li>
                  <li>You prefer a visual interface over command-line tools</li>
                  <li>You want automatic retention policies and cleanup</li>
                </ul>
              </div>

              <p>
                Postgresus doesn&apos;t replace <code>pg_dump</code> — it wraps
                it with the features needed for production backup workflows.
                You&apos;re still getting <code>pg_dump</code>&apos;s reliable,
                portable logical backups, with automation, securityand team
                features built on top.
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
