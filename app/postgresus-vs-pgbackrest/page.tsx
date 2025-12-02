import type { Metadata } from "next";
import Link from "next/link";
import DocsNavbarComponent from "../components/DocsNavbarComponent";
import DocsSidebarComponent from "../components/DocsSidebarComponent";
import DocTableOfContentComponent from "../components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Postgresus vs pgBackRest - PostgreSQL Backup Tools Comparison",
  description:
    "Compare Postgresus and pgBackRest PostgreSQL backup tools. See differences in backup approach, target audience, ease of use, recovery options and when to choose each tool.",
  keywords: [
    "Postgresus vs pgBackRest",
    "PostgreSQL backup comparison",
    "pgBackRest alternative",
    "PostgreSQL backup tools",
    "database backup comparison",
    "pg_dump vs physical backup",
    "self-hosted backup",
    "PostgreSQL PITR",
    "large database backup",
    "DBA backup tools",
  ],
  openGraph: {
    title: "Postgresus vs pgBackRest - PostgreSQL Backup Tools Comparison",
    description:
      "Compare Postgresus and pgBackRest PostgreSQL backup tools. See differences in backup approach, target audience, ease of use, recovery options and when to choose each tool.",
    type: "article",
    url: "https://postgresus.com/postgresus-vs-pgbackrest",
  },
  twitter: {
    card: "summary",
    title: "Postgresus vs pgBackRest - PostgreSQL Backup Tools Comparison",
    description:
      "Compare Postgresus and pgBackRest PostgreSQL backup tools. See differences in backup approach, target audience, ease of use, recovery options and when to choose each tool.",
  },
  alternates: {
    canonical: "https://postgresus.com/postgresus-vs-pgbackrest",
  },
  robots: "index, follow",
};

export default function PostgresusVsPgBackRestPage() {
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
              "Postgresus vs pgBackRest - PostgreSQL Backup Tools Comparison",
            description:
              "A comprehensive comparison of Postgresus and pgBackRest PostgreSQL backup tools, covering backup approach, target audience, ease of use, recovery options and when to choose each tool.",
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
              <h1 id="postgresus-vs-pgbackrest">Postgresus vs pgBackRest</h1>

              <p className="text-lg text-gray-700">
                Postgresus and pgBackRest are both powerful PostgreSQL backup
                tools, but they serve fundamentally different purposes and
                audiences. While Postgresus provides an intuitive web interface
                for individuals, teams and enterprises of all sizes, pgBackRest
                is a specialized command-line tool designed primarily for DBAs
                managing very large databases (500GB+).
              </p>

              <h2 id="quick-comparison">Quick comparison</h2>

              <p>
                Here&apos;s a quick overview of the key differences between
                Postgresus and pgBackRest:
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
                        pgBackRest
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200">
                    <tr className="bg-white hover:bg-gray-50 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Target audience
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        <span className="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800">
                          Individuals, teams, enterprises
                        </span>
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        <span className="inline-flex items-center rounded-full bg-amber-100 px-2.5 py-0.5 text-xs font-medium text-amber-800">
                          DBAs, large enterprises (500GB+ DBs)
                        </span>
                      </td>
                    </tr>
                    <tr className="bg-gray-50 hover:bg-gray-100 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Interface
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Web UI
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Command-line, config files
                      </td>
                    </tr>
                    <tr className="bg-white hover:bg-gray-50 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Backup type
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Logical (full backup)
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Physical (file-level)
                      </td>
                    </tr>
                    <tr className="bg-gray-50 hover:bg-gray-100 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Recovery options
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Restore to any hour or day
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        WAL-based PITR
                      </td>
                    </tr>
                    <tr className="bg-white hover:bg-gray-50 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Parallel operations
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
                            Parallel restores
                          </span>
                        </div>
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
                            Parallel backup & restore
                          </span>
                        </div>
                      </td>
                    </tr>
                    <tr className="bg-gray-50 hover:bg-gray-100 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Incremental backups
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Full backups with compression
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Block-level incremental
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
                            Workspaces, RBAC, audit logs
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
                          Single user
                        </span>
                      </td>
                    </tr>
                    <tr className="bg-gray-50 hover:bg-gray-100 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Learning curve
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        <span className="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800">
                          Minimal
                        </span>
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        <span className="inline-flex items-center rounded-full bg-amber-100 px-2.5 py-0.5 text-xs font-medium text-amber-800">
                          DBA expertise required
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
                        Manual configuration required
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>

              <h2 id="target-audience">Target audience</h2>

              <p>
                The most significant difference between these tools is who they
                are designed for:
              </p>

              <h3 id="audience-postgresus">Postgresus audience</h3>

              <p>
                Postgresus is built for a broad audience, from individual
                developers to large enterprises:
              </p>

              <ul>
                <li>
                  <strong>Individual developers</strong>: Simple setup and
                  intuitive UI make it easy to protect personal projects without
                  deep PostgreSQL expertise.
                </li>
                <li>
                  <strong>Development teams</strong>: Workspaces, role-based
                  access control and audit logs enable secure collaboration
                  across team members.
                </li>
                <li>
                  <strong>Enterprises</strong>: Scales to meet enterprise needs
                  with comprehensive security, multiple storage destinations and
                  notification channels.
                </li>
              </ul>

              <h3 id="audience-pgbackrest">pgBackRest audience</h3>

              <p>
                pgBackRest is specifically designed for Database Administrators
                (DBAs) managing very large databases:
              </p>

              <ul>
                <li>
                  <strong>Large enterprises with 500GB+ databases</strong>:
                  Block-level incremental backups and parallel processing become
                  essential at this scale.
                </li>
                <li>
                  <strong>Professional DBAs</strong>: Requires deep PostgreSQL
                  knowledge, WAL archiving expertise and command-line
                  proficiency.
                </li>
                <li>
                  <strong>Mission-critical systems</strong>: Where
                  second-precise Point-in-Time Recovery is a strict requirement.
                </li>
              </ul>

              <h2 id="backup-approach">Backup approach</h2>

              <p>
                The tools use fundamentally different backup strategies, each
                with distinct advantages:
              </p>

              <h3 id="backup-postgresus">Postgresus: Logical backups</h3>

              <p>
                Postgresus uses <code>pg_dump</code> for logical backups,
                creating SQL representations of your data:
              </p>

              <ul>
                <li>
                  <strong>Portable</strong>: Backups can be restored to
                  different PostgreSQL versions or even different servers.
                </li>
                <li>
                  <strong>Selective restore</strong>: Restore specific tables or
                  schemas without restoring the entire database.
                </li>
                <li>
                  <strong>Efficient compression</strong>: Uses zstd (level 5)
                  compression, reducing backup sizes by 4-8x with only ~20%
                  runtime overhead.
                </li>
                <li>
                  <strong>Read-only access</strong>: Only requires SELECT
                  permissions, minimizing security risks.
                </li>
              </ul>

              <h3 id="backup-pgbackrest">pgBackRest: Physical backups</h3>

              <p>
                pgBackRest performs file-level (physical) backups of the
                PostgreSQL data directory:
              </p>

              <ul>
                <li>
                  <strong>Block-level incremental</strong>: Only changed blocks
                  are backed up, reducing backup time and storage for very large
                  databases.
                </li>
                <li>
                  <strong>WAL archiving</strong>: Continuous archiving of
                  Write-Ahead Logs enables precise Point-in-Time Recovery.
                </li>
                <li>
                  <strong>Full, differential, incremental</strong>: Multiple
                  backup strategies for different recovery scenarios.
                </li>
                <li>
                  <strong>Optimized for scale</strong>: Designed for databases
                  where logical backups would take too long.
                </li>
              </ul>

              <h2 id="recovery-options">Recovery options</h2>

              <p>
                Both tools offer flexible recovery options, but with different
                granularity:
              </p>

              <h3 id="recovery-postgresus">Postgresus recovery</h3>

              <ul>
                <li>
                  <strong>Restore to any hour or day</strong>: With hourly,
                  daily, weekly or monthly backup schedules, you can restore to
                  any backup point you&apos;ve configured.
                </li>
                <li>
                  <strong>One-click restore</strong>: Download and restore
                  backups directly from the web interface.
                </li>
                <li>
                  <strong>Parallel restores</strong>: Utilize multiple CPU cores
                  to speed up restoration of large backups.
                </li>
                <li>
                  <strong>Cross-version compatibility</strong>: Restore backups
                  to different PostgreSQL versions when needed.
                </li>
              </ul>

              <h3 id="recovery-pgbackrest">pgBackRest recovery</h3>

              <ul>
                <li>
                  <strong>Point-in-Time Recovery (PITR)</strong>: Restore to any
                  specific second using WAL replay.
                </li>
                <li>
                  <strong>Parallel restore</strong>: Multi-threaded restoration
                  for faster recovery of large databases.
                </li>
                <li>
                  <strong>Delta restore</strong>: Only restore changed files,
                  reducing recovery time.
                </li>
                <li>
                  <strong>Standby creation</strong>: Create PostgreSQL replicas
                  from backups.
                </li>
              </ul>

              <div className="rounded-lg border border-blue-200 bg-blue-50 p-4 my-6">
                <p className="text-blue-900 m-0">
                  <strong>Note:</strong> For most applications, restoring to the
                  nearest hour or day (as Postgresus provides) is sufficient.
                  Second-precise PITR is typically only required for
                  mission-critical financial or transactional systems where
                  every transaction must be recoverable.
                </p>
              </div>

              <h2 id="ease-of-use">Ease of use</h2>

              <p>
                The tools differ dramatically in their approach to user
                experience:
              </p>

              <h3 id="ease-postgresus">Postgresus user experience</h3>

              <ul>
                <li>
                  <strong>Web interface</strong>: Point-and-click configuration
                  for all backup settings. No command-line required.
                </li>
                <li>
                  <strong>2-minute installation</strong>: One-line cURL script
                  or simple Docker command gets you running immediately.
                </li>
                <li>
                  <strong>Visual monitoring</strong>: Dashboard shows backup
                  status, health checks and history at a glance.
                </li>
                <li>
                  <strong>Built-in notifications</strong>: Configure Slack,
                  Teams, Telegram, Email or webhook alerts directly in the UI.
                </li>
                <li>
                  <strong>No PostgreSQL expertise required</strong>: Designed
                  for developers who want reliable backups without becoming
                  database experts.
                </li>
              </ul>

              <h3 id="ease-pgbackrest">pgBackRest user experience</h3>

              <ul>
                <li>
                  <strong>Command-line interface</strong>: All operations
                  performed via terminal commands.
                </li>
                <li>
                  <strong>Configuration files</strong>: Requires manual editing
                  of INI-style configuration files.
                </li>
                <li>
                  <strong>WAL archiving setup</strong>: Must configure
                  PostgreSQL&apos;s <code>archive_command</code> and related
                  settings.
                </li>
                <li>
                  <strong>Steep learning curve</strong>: Requires understanding
                  of PostgreSQL internals, WAL mechanics and backup strategies.
                </li>
                <li>
                  <strong>DBA expertise expected</strong>: Documentation assumes
                  familiarity with database administration concepts.
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

              <h2 id="team-features">Team features</h2>

              <p>
                For organizations with multiple team members managing backups:
              </p>

              <h3 id="team-postgresus">Postgresus team capabilities</h3>

              <ul>
                <li>
                  <strong>Workspaces</strong>: Organize databases, notifiers and
                  storages by project or team. Users only see workspaces
                  they&apos;re invited to.
                </li>
                <li>
                  <strong>Role-based access control</strong>: Assign viewer,
                  editor or admin permissions to control what each team member
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

              <h3 id="team-pgbackrest">pgBackRest team capabilities</h3>

              <p>
                pgBackRest is a command-line tool without built-in team
                features:
              </p>

              <ul>
                <li>No user management or access control</li>
                <li>No audit logging of operations</li>
                <li>Team coordination requires external tools and processes</li>
                <li>
                  Access controlled via OS-level permissions on configuration
                  files
                </li>
              </ul>

              <p>
                <Link
                  href="/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Learn more about Postgresus access management →
                </Link>
              </p>

              <h2 id="security">Security</h2>

              <p>Both tools provide robust security features:</p>

              <h3 id="security-postgresus">Postgresus security</h3>

              <ul>
                <li>
                  <strong>AES-256-GCM encryption</strong>: All passwords, tokens
                  and credentials are encrypted. The encryption key is stored
                  separately from the database.
                </li>
                <li>
                  <strong>Unique backup encryption</strong>: Each backup file is
                  encrypted with a unique key derived from master key, backup ID
                  and random salt.
                </li>
                <li>
                  <strong>Read-only database access</strong>: Enforces SELECT
                  permissions only, preventing data corruption even if
                  compromised.
                </li>
              </ul>

              <h3 id="security-pgbackrest">pgBackRest security</h3>

              <ul>
                <li>
                  <strong>Repository encryption</strong>: Backup repositories
                  can be encrypted with AES-256.
                </li>
                <li>
                  <strong>TLS/SSH transport</strong>: Secure communication for
                  remote operations.
                </li>
                <li>
                  <strong>Checksum verification</strong>: Validates backup
                  integrity during creation and restore.
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

              <h2 id="storage-options">Storage options</h2>

              <p>
                Both tools support multiple storage destinations, with
                Postgresus offering more consumer-friendly options:
              </p>

              <h3 id="storage-postgresus">Postgresus storage</h3>

              <ul>
                <li>Local storage</li>
                <li>Amazon S3 and S3-compatible services</li>
                <li>Google Drive</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (Network-attached storage)</li>
                <li>Dropbox</li>
              </ul>

              <h3 id="storage-pgbackrest">pgBackRest storage</h3>

              <ul>
                <li>Local storage (POSIX, CIFS)</li>
                <li>Amazon S3 and S3-compatible services</li>
                <li>Azure Blob Storage</li>
                <li>Google Cloud Storage</li>
                <li>NAS (Network-attached storage)</li>
                <li>SFTP</li>
              </ul>

              <p>
                <Link
                  href="/storages"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  View all Postgresus storage options →
                </Link>
              </p>

              <h2 id="notifications">Notifications</h2>

              <p>Staying informed about backup status:</p>

              <h3 id="notifications-postgresus">Postgresus notifications</h3>

              <p>Built-in support for multiple notification channels:</p>

              <ul>
                <li>Slack</li>
                <li>Discord</li>
                <li>Telegram</li>
                <li>Microsoft Teams</li>
                <li>Email</li>
                <li>Webhooks</li>
              </ul>

              <h3 id="notifications-pgbackrest">pgBackRest notifications</h3>

              <p>
                pgBackRest does not have built-in notification support.
                Notifications require:
              </p>

              <ul>
                <li>Custom scripting around backup commands</li>
                <li>External monitoring tools integration</li>
                <li>Manual log parsing and alerting setup</li>
              </ul>

              <p>
                <Link
                  href="/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  View all Postgresus notification channels →
                </Link>
              </p>

              <h2 id="conclusion">Conclusion</h2>

              <p>
                Postgresus and pgBackRest serve different needs in the
                PostgreSQL backup ecosystem. The right choice depends on your
                database size, team structure and technical requirements.
              </p>

              <div className="rounded-lg border border-blue-200 bg-blue-50 p-4 my-6">
                <p className="text-blue-900 m-0">
                  <strong>Choose Postgresus if:</strong>
                </p>
                <ul className="text-blue-900 mb-0">
                  <li>
                    You&apos;re an individual developer, team or enterprise
                    looking for an intuitive backup solution
                  </li>
                  <li>You prefer a web interface over command-line tools</li>
                  <li>
                    You need team collaboration features (workspaces, RBAC,
                    audit logs)
                  </li>
                  <li>
                    You want built-in notifications to Slack, Teams, Telegram
                    etc.
                  </li>
                  <li>
                    Restoring to any hour or day meets your recovery
                    requirements
                  </li>
                  <li>You need parallel restores for faster recovery times</li>
                  <li>
                    You want quick setup with minimal PostgreSQL expertise
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-gray-200 bg-gray-50 p-4 my-6">
                <p className="text-gray-900 m-0">
                  <strong>Choose pgBackRest if:</strong>
                </p>
                <ul className="text-gray-900 mb-0">
                  <li>
                    You&apos;re a DBA managing very large databases (500GB+)
                  </li>
                  <li>
                    You require second-precise Point-in-Time Recovery for
                    mission-critical systems
                  </li>
                  <li>
                    Block-level incremental backups are essential for your scale
                  </li>
                  <li>
                    You&apos;re comfortable with command-line tools and
                    PostgreSQL internals
                  </li>
                  <li>
                    Your organization has dedicated DBA expertise available
                  </li>
                </ul>
              </div>

              <p>
                For most use cases, from individual projects to enterprise
                deployments, Postgresus provides the right balance of power and
                usability. pgBackRest remains the specialized choice for DBAs
                managing very large databases where its advanced features become
                necessary - it&apos;s the best tool for such cases.
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
