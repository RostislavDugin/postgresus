import type { Metadata } from "next";
import Link from "next/link";
import DocsNavbarComponent from "../components/DocsNavbarComponent";
import DocsSidebarComponent from "../components/DocsSidebarComponent";
import DocTableOfContentComponent from "../components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Postgresus vs PgBackWeb - PostgreSQL Backup Tools Comparison",
  description:
    "Compare Postgresus and PgBackWeb PostgreSQL backup tools. See differences in features, security, team support, storage options, notifications and ease of use.",
  keywords: [
    "Postgresus vs PgBackWeb",
    "PostgreSQL backup comparison",
    "PgBackWeb alternative",
    "PostgreSQL backup tools",
    "database backup comparison",
    "pg_dump GUI",
    "self-hosted backup",
    "PostgreSQL backup security",
  ],
  openGraph: {
    title: "Postgresus vs PgBackWeb - PostgreSQL Backup Tools Comparison",
    description:
      "Compare Postgresus and PgBackWeb PostgreSQL backup tools. See differences in features, security, team support, storage options, notifications and ease of use.",
    type: "article",
    url: "https://postgresus.com/postgresus-vs-pgbackweb",
  },
  twitter: {
    card: "summary",
    title: "Postgresus vs PgBackWeb - PostgreSQL Backup Tools Comparison",
    description:
      "Compare Postgresus and PgBackWeb PostgreSQL backup tools. See differences in features, security, team support, storage options, notifications and ease of use.",
  },
  alternates: {
    canonical: "https://postgresus.com/postgresus-vs-pgbackweb",
  },
  robots: "index, follow",
};

export default function PostgresusVsPgBackWebPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline:
              "Postgresus vs PgBackWeb - PostgreSQL Backup Tools Comparison",
            description:
              "A comprehensive comparison of Postgresus and PgBackWeb PostgreSQL backup tools, covering features, security, team support, storage options and ease of use.",
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
              <h1 id="postgresus-vs-pgbackweb">Postgresus vs PgBackWeb</h1>

              <p className="text-lg text-gray-700">
                Both Postgresus and PgBackWeb are open-source tools designed to
                simplify PostgreSQL backup management through web interfaces.
                While they share the common goal of making backups more
                accessible, they differ significantly in features, security,
                team support and ease of use.
              </p>

              <h2 id="quick-comparison">Quick comparison</h2>

              <p>
                Here&apos;s a quick overview of the key differences between
                Postgresus and PgBackWeb:
              </p>

              <div className="not-prose overflow-x-auto my-6">
                <table className="w-full border-collapse rounded-lg overflow-hidden shadow-sm border border-gray-200">
                  <thead>
                    <tr className="bg-gray-100 border-b border-gray-200">
                      <th className="px-4 py-3 text-center font-semibold text-gray-900">
                        Feature
                      </th>
                      <th className="px-4 py-3 text-center font-semibold text-gray-900">
                        Postgresus
                      </th>
                      <th className="px-4 py-3 text-center font-semibold text-gray-900">
                        PgBackWeb
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200">
                    <tr className="bg-white hover:bg-gray-50 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        License
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        <span className="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800">
                          Apache 2.0
                        </span>
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        <span className="inline-flex items-center rounded-full bg-amber-100 px-2.5 py-0.5 text-xs font-medium text-amber-800">
                          AGPL-3.0
                        </span>
                      </td>
                    </tr>
                    <tr className="bg-gray-50 hover:bg-gray-100 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Storage options
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Local, S3, Google Drive, Cloudflare R2, Azure, NAS,
                        Dropbox
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Local, S3-compatible only
                      </td>
                    </tr>
                    <tr className="bg-white hover:bg-gray-50 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Notifications
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Slack, Discord, Telegram, Teams, Email, Webhooks
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Webhooks only
                      </td>
                    </tr>
                    <tr className="bg-gray-50 hover:bg-gray-100 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Security
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        <div className="flex flex-col gap-1 items-center">
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
                            AES-256-GCM encryption
                          </span>
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
                            Unique backup keys
                          </span>
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
                            Read-only enforcement
                          </span>
                        </div>
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
                          PGP encryption
                        </span>
                      </td>
                    </tr>
                    <tr className="bg-white hover:bg-gray-50 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Team features
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        <div className="flex flex-col gap-1 items-center">
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
                            Workspaces
                          </span>
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
                            Role-based access
                          </span>
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
                            Audit logs
                          </span>
                        </div>
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
                          Not available
                        </span>
                      </td>
                    </tr>
                    <tr className="bg-gray-50 hover:bg-gray-100 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Health monitoring
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
                          Built-in
                        </span>
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
                          Not available
                        </span>
                      </td>
                    </tr>
                    <tr className="bg-white hover:bg-gray-50 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Installation
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        <span className="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800">
                          One-line script or Docker
                        </span>
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Manual Docker setup
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>

              <h2 id="backup-features">Backup features</h2>

              <p>Both tools support scheduled backups with flexible timing:</p>

              <ul>
                <li>
                  <strong>Postgresus</strong>: Supports hourly, daily, weekly
                  and monthly schedules with precise timing (e.g. 4 AM).
                  Implements{" "}
                  <strong>balanced compression using zstd (level 5)</strong>,
                  reducing backup sizes by 4-8x with only ~20% runtime overhead.
                  This is significantly more efficient than gzip.
                </li>
                <li>
                  <strong>PgBackWeb</strong>: Supports cron-based scheduling for
                  backup execution. Uses gzip compression for backups which is
                  slower and less efficient than zstd.
                </li>
              </ul>

              <h2 id="storage-options">Storage options</h2>

              <p>
                Storage flexibility is crucial for backup strategies.
                Here&apos;s how the two tools compare:
              </p>

              <ul>
                <li>
                  <strong>Postgresus</strong>: Supports a wide range of storage
                  destinations:
                  <ul>
                    <li>Local storage</li>
                    <li>Amazon S3 and S3-compatible services</li>
                    <li>Google Drive</li>
                    <li>Cloudflare R2</li>
                    <li>Azure Blob Storage</li>
                    <li>NAS (Network-attached storage)</li>
                  </ul>
                </li>
                <li>
                  <strong>PgBackWeb</strong>: Limited to local storage and
                  S3-compatible storage only.
                </li>
              </ul>

              <p>
                <Link
                  href="/storages"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  View all Postgresus storage options →
                </Link>
              </p>

              <h2 id="security">Security</h2>

              <p>
                Security is a critical aspect of backup management. Postgresus
                implements enterprise-grade security on three levels:
              </p>

              <h3 id="security-postgresus">Postgresus security model</h3>

              <ol>
                <li>
                  <strong>Sensitive data encryption</strong>: All passwords,
                  tokens and credentials are encrypted with AES-256-GCM. The
                  encryption key is stored separately from the database, so even
                  if the database is compromised, sensitive data remains
                  protected.
                </li>
                <li>
                  <strong>Backup encryption</strong>: Each backup file is
                  encrypted with a unique key derived from the master key,
                  backup ID and random salt. Even if someone gains access to
                  your cloud storage, they cannot read the backups without your
                  encryption key.
                </li>
                <li>
                  <strong>Read-only database access</strong>: Postgresus
                  enforces read-only access by checking role-level,
                  database-level and table-level permissions. It only requires
                  SELECT permissions and will warn you if write privileges are
                  detected. This prevents data corruption even if Postgresus is
                  compromised.
                </li>
              </ol>

              <h3 id="security-pgbackweb">PgBackWeb security model</h3>

              <ul>
                <li>
                  <strong>PGP encryption</strong>: PgBackWeb offers PGP
                  encryption for backup files.
                </li>
                <li>
                  <strong>No read-only enforcement</strong>: PgBackWeb does not
                  enforce or verify read-only database access which means
                  backups may be created with users that have write permissions.
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

              <h2 id="notifications">Notifications</h2>

              <p>
                Staying informed about backup status is essential for reliable
                operations:
              </p>

              <ul>
                <li>
                  <strong>Postgresus</strong>: Provides real-time notifications
                  through multiple channels:
                  <ul>
                    <li>Slack</li>
                    <li>Discord</li>
                    <li>Telegram</li>
                    <li>Microsoft Teams</li>
                    <li>Email</li>
                    <li>Webhooks</li>
                  </ul>
                </li>
                <li>
                  <strong>PgBackWeb</strong>: Supports webhooks only for
                  notifications. To receive alerts via Slack, Telegram or other
                  platforms you need to set up additional middleware or
                  services.
                </li>
              </ul>

              <p>
                <Link
                  href="/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  View all Postgresus notification channels →
                </Link>
              </p>

              <h2 id="team-features">Team features</h2>

              <p>
                For organizations and DevOps teams, collaboration features are
                essential. This is where Postgresus significantly outshines
                PgBackWeb:
              </p>

              <h3 id="team-postgresus">Postgresus team capabilities</h3>

              <ul>
                <li>
                  <strong>Workspaces</strong>: Group databases, notifiers and
                  storages for different projects or teams. Users only see
                  workspaces they&apos;re invited to.
                </li>
                <li>
                  <strong>Role-based access control</strong>: Permission levels
                  to control what each team member can do within workspaces.
                </li>
                <li>
                  <strong>Audit logs</strong>: Track all system activities and
                  changes made by users. Essential for security compliance and
                  team accountability.
                </li>
              </ul>

              <h3 id="team-pgbackweb">PgBackWeb team capabilities</h3>

              <p>
                PgBackWeb does not have built-in user management, workspaces or
                audit logs. It&apos;s designed primarily for single-user
                scenarios.
              </p>

              <p>
                <Link
                  href="/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Learn more about Postgresus access management →
                </Link>
              </p>

              <h2 id="ease-of-use">Ease of use</h2>

              <p>
                <strong>
                  Postgresus is designed to be significantly easier to use
                </strong>{" "}
                than PgBackWeb, with a focus on intuitive UX and minimal setup
                time:
              </p>

              <h3 id="ease-postgresus">Postgresus user experience</h3>

              <ul>
                <li>
                  <strong>Easy installation</strong>: Use Docker directly or run
                  a one-line script that installs Docker (if needed), sets up
                  Postgresus and configures automatic startup. Total time: ~2
                  minutes.
                </li>
                <li>
                  <strong>Intuitive web interface</strong>: Designer-polished UI
                  that guides you through backup configuration step by step. No
                  PostgreSQL expertise required.
                </li>
                <li>
                  <strong>Dark and light themes</strong>: Choose the look that
                  suits your workflow.
                </li>
                <li>
                  <strong>Mobile adaptive</strong>: Check your backups from
                  anywhere on any device.
                </li>
                <li>
                  <strong>Built-in health monitoring</strong>: Configurable
                  health checks with visual availability charts.
                </li>
                <li>
                  <strong>One-click restore</strong>: Download and restore from
                  any backup with a single click.
                </li>
              </ul>

              <h3 id="ease-pgbackweb">PgBackWeb user experience</h3>

              <ul>
                <li>
                  <strong>Manual Docker setup</strong>: Requires configuring
                  environment variables and setting up an external PostgreSQL
                  database for configuration storage.
                </li>
                <li>
                  <strong>Basic web interface</strong>: Functional but less
                  polished UI compared to Postgresus. Dark theme available.
                </li>
                <li>
                  <strong>No health monitoring</strong>: Database availability
                  monitoring must be set up separately.
                </li>
              </ul>

              <h2 id="installation">Installation and deployment</h2>

              <h3 id="install-postgresus">Installing Postgresus</h3>

              <p>
                Postgresus offers three installation methods, with the automated
                script being the quickest:
              </p>

              <ul>
                <li>
                  <strong>Automated script (recommended)</strong>: One-line cURL
                  command that installs Docker, sets up Postgresus and
                  configures automatic startup.
                </li>
                <li>
                  <strong>Docker run</strong>: Single command to start
                  Postgresus with embedded PostgreSQL.
                </li>
                <li>
                  <strong>Docker Compose</strong>: For more control over the
                  deployment.
                </li>
              </ul>

              <p>
                <Link
                  href="/installation"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  View Postgresus installation guide →
                </Link>
              </p>

              <h3 id="install-pgbackweb">Installing PgBackWeb</h3>

              <p>
                PgBackWeb requires Docker and manual configuration of
                environment variables. You also need to set up an external
                PostgreSQL database for storing PgBackWeb&apos;s configuration.
              </p>

              <h2 id="licensing">Licensing</h2>

              <p>
                The licensing model can significantly impact how you can use and
                modify the software:
              </p>

              <ul>
                <li>
                  <strong>Postgresus (Apache 2.0)</strong>: Permissive license
                  that allows unrestricted commercial use, modification and
                  distribution. You can use Postgresus in proprietary projects
                  without any licensing concerns.
                </li>
                <li>
                  <strong>PgBackWeb (AGPL-3.0)</strong>: Copyleft license that
                  requires any derivative works or modifications to also be
                  open-source under AGPL-3.0. If you modify PgBackWeb and
                  provide it as a service you must release your modifications.
                </li>
              </ul>

              <h2 id="conclusion">Conclusion</h2>

              <p>
                Both Postgresus and PgBackWeb are capable PostgreSQL backup
                tools, but they serve different needs:
              </p>

              <div className="rounded-lg border border-blue-200 bg-blue-50 p-4 my-6">
                <p className="text-blue-900 m-0">
                  <strong>Choose Postgresus if you need:</strong>
                </p>
                <ul className="text-blue-900 mb-0">
                  <li>Enterprise-grade security with 3-level protection</li>
                  <li>Team collaboration with workspaces and audit logs</li>
                  <li>
                    Multiple storage destinations (Google Drive, Azure etc.)
                  </li>
                  <li>Built-in notifications to Slack, Teams, Telegram etc.</li>
                  <li>Quick installation with one-line script or Docker</li>
                  <li>Intuitive modern UI with minimal learning curve</li>
                  <li>Permissive Apache 2.0 license for commercial use</li>
                </ul>
              </div>

              <div className="rounded-lg border border-gray-200 bg-gray-50 p-4 my-6">
                <p className="text-gray-900 m-0">
                  <strong>Choose PgBackWeb if you need:</strong>
                </p>
                <ul className="text-gray-900 mb-0">
                  <li>Simple backup solution for single-user scenarios</li>
                  <li>Only local or S3 storage</li>
                  <li>Webhook-only notifications are sufficient</li>
                  <li>AGPL-3.0 license is acceptable for your use case</li>
                </ul>
              </div>

              <p>
                For most users, especially teams and organizations requiring
                robust security, multiple storage options and comprehensive
                notification channels,{" "}
                <strong>Postgresus is the recommended choice</strong>.
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
