import type { Metadata } from "next";
import Link from "next/link";
import DocsNavbarComponent from "../components/DocsNavbarComponent";
import DocsSidebarComponent from "../components/DocsSidebarComponent";
import DocTableOfContentComponent from "../components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Postgresus vs Barman - PostgreSQL Backup Tools Comparison",
  description:
    "Compare Postgresus and Barman PostgreSQL backup tools. See differences in backup approach, PITR capabilities, ease of use, team features and when to choose each tool.",
  keywords: [
    "Postgresus vs Barman",
    "PostgreSQL backup comparison",
    "Barman alternative",
    "PostgreSQL backup tools",
    "database backup comparison",
    "pg_dump vs physical backup",
    "self-hosted backup",
    "PostgreSQL PITR",
    "WAL archiving",
    "disaster recovery PostgreSQL",
  ],
  openGraph: {
    title: "Postgresus vs Barman - PostgreSQL Backup Tools Comparison",
    description:
      "Compare Postgresus and Barman PostgreSQL backup tools. See differences in backup approach, PITR capabilities, ease of use, team features and when to choose each tool.",
    type: "article",
    url: "https://postgresus.com/postgresus-vs-barman",
  },
  twitter: {
    card: "summary",
    title: "Postgresus vs Barman - PostgreSQL Backup Tools Comparison",
    description:
      "Compare Postgresus and Barman PostgreSQL backup tools. See differences in backup approach, PITR capabilities, ease of use, team features and when to choose each tool.",
  },
  alternates: {
    canonical: "https://postgresus.com/postgresus-vs-barman",
  },
  robots: "index, follow",
};

export default function PostgresusVsBarmanPage() {
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
              "Postgresus vs Barman - PostgreSQL Backup Tools Comparison",
            description:
              "A comprehensive comparison of Postgresus and Barman PostgreSQL backup tools, covering backup approach, PITR capabilities, ease of use, team features and when to choose each tool.",
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
              <h1 id="postgresus-vs-barman">Postgresus vs Barman</h1>

              <p className="text-lg text-gray-700">
                Postgresus and Barman are both PostgreSQL backup tools, but they
                take fundamentally different approaches. Postgresus provides an
                intuitive web interface for logical backups with team
                collaboration features, while Barman (Backup and Recovery
                Manager) is a command-line tool focused on physical backups and
                Point-in-Time Recovery (PITR) for enterprise disaster recovery
                scenarios.
              </p>

              <h2 id="quick-comparison">Quick comparison</h2>

              <p>
                Here&apos;s a quick overview of the key differences between
                Postgresus and Barman:
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
                        Barman
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
                          DBAs, enterprises requiring PITR
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
                        Command-line only
                      </td>
                    </tr>
                    <tr className="bg-white hover:bg-gray-50 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Backup type
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Logical (pg_dump)
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
                        WAL-based PITR (second-precise)
                      </td>
                    </tr>
                    <tr className="bg-white hover:bg-gray-50 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Incremental backups
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Full backups with compression
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        rsync-based incremental
                      </td>
                    </tr>
                    <tr className="bg-gray-50 hover:bg-gray-100 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Multi-server management
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Per-database scheduling
                      </td>
                      <td className="px-4 py-3 text-center text-gray-700">
                        Centralized backup server
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
                          OS-level permissions only
                        </span>
                      </td>
                    </tr>
                    <tr className="bg-gray-50 hover:bg-gray-100 transition-colors">
                      <td className="px-4 py-3 text-center font-medium text-gray-900">
                        Notifications
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
                          Slack, Teams, Telegram, Email
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
                          Requires custom scripting
                        </span>
                      </td>
                    </tr>
                    <tr className="bg-white hover:bg-gray-50 transition-colors">
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
                    <tr className="bg-gray-50 hover:bg-gray-100 transition-colors">
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

              <h3 id="audience-barman">Barman audience</h3>

              <p>
                Barman is specifically designed for Database Administrators
                (DBAs) managing enterprise PostgreSQL infrastructure:
              </p>

              <ul>
                <li>
                  <strong>Enterprise DBAs</strong>: Professionals who need
                  centralized backup management for multiple PostgreSQL servers
                  from a single location.
                </li>
                <li>
                  <strong>Disaster recovery specialists</strong>: Teams
                  requiring second-precise Point-in-Time Recovery for
                  mission-critical systems.
                </li>
                <li>
                  <strong>
                    Organizations with strict RPO/RTO requirements
                  </strong>
                  : Where Recovery Point Objective demands minimal data loss and
                  WAL-based recovery is essential.
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
                  <strong>Efficient compression</strong>: Uses zstd (level 5)
                  compression, reducing backup sizes by 4-8x with only ~20%
                  runtime overhead.
                </li>
                <li>
                  <strong>Read-only access</strong>: Only requires SELECT
                  permissions, minimizing security risks.
                </li>
              </ul>

              <h3 id="backup-barman">Barman: Physical backups</h3>

              <p>
                Barman performs file-level (physical) backups of the PostgreSQL
                data directory:
              </p>

              <ul>
                <li>
                  <strong>Full cluster backup</strong>: Captures the entire
                  database cluster at the file system level using rsync or
                  pg_basebackup.
                </li>
                <li>
                  <strong>WAL archiving</strong>: Continuously archives
                  Write-Ahead Logs for Point-in-Time Recovery.
                </li>
                <li>
                  <strong>Incremental with rsync</strong>: Uses rsync to
                  transfer only changed files, reducing backup time and network
                  usage.
                </li>
                <li>
                  <strong>Streaming replication integration</strong>: Can
                  receive WAL files via streaming replication protocol for
                  real-time archiving.
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

              <h3 id="recovery-barman">Barman recovery</h3>

              <ul>
                <li>
                  <strong>Point-in-Time Recovery (PITR)</strong>: Restore to any
                  specific second using WAL replay, minimizing data loss.
                </li>
                <li>
                  <strong>Full cluster restore</strong>: Restore the entire
                  database cluster to a specific point in time.
                </li>
                <li>
                  <strong>Remote recovery</strong>: Recover databases to remote
                  servers over SSH.
                </li>
                <li>
                  <strong>Standby creation</strong>: Create PostgreSQL replicas
                  from backups for high availability setups.
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

              <h3 id="ease-barman">Barman user experience</h3>

              <ul>
                <li>
                  <strong>Command-line interface</strong>: All operations
                  performed via terminal commands like{" "}
                  <code>barman backup</code>, <code>barman recover</code>.
                </li>
                <li>
                  <strong>Configuration files</strong>: Requires manual editing
                  of INI-style configuration files for each server.
                </li>
                <li>
                  <strong>WAL archiving setup</strong>: Must configure
                  PostgreSQL&apos;s <code>archive_command</code> or streaming
                  replication settings.
                </li>
                <li>
                  <strong>SSH key management</strong>: Requires setting up SSH
                  keys between Barman server and PostgreSQL servers.
                </li>
                <li>
                  <strong>DBA expertise expected</strong>: Documentation assumes
                  familiarity with PostgreSQL internals and WAL mechanics.
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

              <h3 id="team-barman">Barman team capabilities</h3>

              <p>
                Barman is a command-line tool without built-in team features:
              </p>

              <ul>
                <li>No user management or access control</li>
                <li>No audit logging of operations</li>
                <li>Team coordination requires external tools and processes</li>
                <li>Access controlled via OS-level permissions and SSH keys</li>
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

              <p>
                Both tools provide security features, but with different
                approaches:
              </p>

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

              <h3 id="security-barman">Barman security</h3>

              <ul>
                <li>
                  <strong>SSH-based communication</strong>: Uses SSH for secure
                  communication between Barman server and PostgreSQL servers.
                </li>
                <li>
                  <strong>No built-in encryption</strong>: Barman does not
                  provide built-in backup encryption. External tools or
                  encrypted storage must be used.
                </li>
                <li>
                  <strong>OS-level security</strong>: Relies on file system
                  permissions and SSH key management for access control.
                </li>
                <li>
                  <strong>Checksum verification</strong>: Validates backup
                  integrity using checksums.
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

              <p>The tools support different storage destinations:</p>

              <h3 id="storage-postgresus">Postgresus storage</h3>

              <p>Consumer-friendly options for various use cases:</p>

              <ul>
                <li>Local storage</li>
                <li>Amazon S3 and S3-compatible services</li>
                <li>Google Drive</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (Network-attached storage)</li>
                <li>Dropbox</li>
              </ul>

              <h3 id="storage-barman">Barman storage</h3>

              <p>Enterprise-focused storage options:</p>

              <ul>
                <li>Local storage (POSIX file systems)</li>
                <li>Amazon S3 and S3-compatible object storage</li>
                <li>
                  Geographical redundancy via Barman-to-Barman replication
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

              <h3 id="notifications-barman">Barman notifications</h3>

              <p>
                Barman does not have built-in notification support.
                Notifications require:
              </p>

              <ul>
                <li>Custom scripting around backup commands</li>
                <li>External monitoring tools integration</li>
                <li>Manual log parsing and alerting setup</li>
                <li>
                  Integration with tools like Nagios, Zabbix or custom solutions
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

              <h2 id="multi-server-management">Multi-server management</h2>

              <p>
                Both tools can manage backups for multiple PostgreSQL servers,
                but with different approaches:
              </p>

              <h3 id="multi-postgresus">Postgresus approach</h3>

              <ul>
                <li>
                  <strong>Per-database scheduling</strong>: Each database can
                  have its own backup schedule and storage destination.
                </li>
                <li>
                  <strong>Workspace organization</strong>: Group related
                  databases into workspaces for easier management.
                </li>
                <li>
                  <strong>Unified dashboard</strong>: View all database backups
                  and their status in a single web interface.
                </li>
              </ul>

              <h3 id="multi-barman">Barman approach</h3>

              <ul>
                <li>
                  <strong>Centralized backup server</strong>: A dedicated Barman
                  server manages backups for multiple PostgreSQL instances.
                </li>
                <li>
                  <strong>Configuration per server</strong>: Each PostgreSQL
                  server requires its own configuration file on the Barman
                  server.
                </li>
                <li>
                  <strong>Geo-redundancy</strong>: Barman servers can replicate
                  to other Barman servers for geographical redundancy.
                </li>
              </ul>

              <h2 id="conclusion">Conclusion</h2>

              <p>
                Postgresus and Barman serve different needs in the PostgreSQL
                backup ecosystem. The right choice depends on your recovery
                requirements, team structure and technical expertise.
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
                  <li>
                    You want quick setup with minimal PostgreSQL expertise
                  </li>
                  <li>Built-in backup encryption is important to you</li>
                </ul>
              </div>

              <div className="rounded-lg border border-gray-200 bg-gray-50 p-4 my-6">
                <p className="text-gray-900 m-0">
                  <strong>Choose Barman if:</strong>
                </p>
                <ul className="text-gray-900 mb-0">
                  <li>
                    You require second-precise Point-in-Time Recovery for
                    mission-critical systems
                  </li>
                  <li>
                    You need centralized management of multiple PostgreSQL
                    servers from a dedicated backup server
                  </li>
                  <li>
                    WAL archiving and streaming replication integration is
                    essential
                  </li>
                  <li>
                    You&apos;re comfortable with command-line tools and
                    PostgreSQL internals
                  </li>
                  <li>
                    Your organization has dedicated DBA expertise available
                  </li>
                  <li>You need Barman-to-Barman geographical redundancy</li>
                </ul>
              </div>

              <p>
                For most use cases, from individual projects to enterprise
                deployments, Postgresus provides the right balance of power and
                usability. Barman remains the specialized choice for
                organizations with strict PITR requirements and dedicated DBA
                teams — it excels in enterprise disaster recovery scenarios
                where second-precise recovery is non-negotiable.
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
