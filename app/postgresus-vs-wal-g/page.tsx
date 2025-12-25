import type { Metadata } from "next";
import Link from "next/link";
import DocsNavbarComponent from "../components/DocsNavbarComponent";
import DocsSidebarComponent from "../components/DocsSidebarComponent";
import DocTableOfContentComponent from "../components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Postgresus vs WAL-G - PostgreSQL Backup Tools Comparison",
  description:
    "Compare Postgresus and WAL-G PostgreSQL backup tools. See differences in backup approach, multi-database support, ease of use, team features and when to choose each tool.",
  keywords: [
    "Postgresus vs WAL-G",
    "PostgreSQL backup comparison",
    "WAL-G alternative",
    "PostgreSQL backup tools",
    "database backup comparison",
    "pg_dump vs WAL archiving",
    "self-hosted backup",
    "PostgreSQL PITR",
    "WAL archiving",
    "multi-database backup",
  ],
  openGraph: {
    title: "Postgresus vs WAL-G - PostgreSQL Backup Tools Comparison",
    description:
      "Compare Postgresus and WAL-G PostgreSQL backup tools. See differences in backup approach, multi-database support, ease of use, team features and when to choose each tool.",
    type: "article",
    url: "https://postgresus.com/postgresus-vs-wal-g",
  },
  twitter: {
    card: "summary",
    title: "Postgresus vs WAL-G - PostgreSQL Backup Tools Comparison",
    description:
      "Compare Postgresus and WAL-G PostgreSQL backup tools. See differences in backup approach, multi-database support, ease of use, team features and when to choose each tool.",
  },
  alternates: {
    canonical: "https://postgresus.com/postgresus-vs-wal-g",
  },
  robots: "index, follow",
};

export default function PostgresusVsWalGPage() {
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
              "Postgresus vs WAL-G - PostgreSQL Backup Tools Comparison",
            description:
              "A comprehensive comparison of Postgresus and WAL-G PostgreSQL backup tools, covering backup approach, multi-database support, ease of use, team features and when to choose each tool.",
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
              <h1 id="postgresus-vs-wal-g">Postgresus vs WAL-G</h1>

              <p className="text-lg text-gray-400">
                Postgresus and WAL-G are both capable backup tools that support
                PostgreSQL, but they take fundamentally different approaches.
                Postgresus focuses on comprehensive backup management with an
                intuitive web interface, while WAL-G is a command-line tool that
                supports multiple database systems including PostgreSQL, MySQL,
                MS SQL, MongoDB and others. If you need a user-friendly solution
                for managing backups across multiple databases, Postgresus
                offers a more streamlined experience.
              </p>

              <h2 id="quick-comparison">Quick comparison</h2>

              <p>
                Here&apos;s a quick overview of the key differences between
                Postgresus and WAL-G:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Feature</th>
                    <th>Postgresus</th>
                    <th>WAL-G</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Database focus</td>
                    <td data-label="Postgresus">PostgreSQL only</td>
                    <td data-label="WAL-G">
                      Multi-database (PostgreSQL, MySQL, MS SQL, etc.)
                    </td>
                  </tr>
                  <tr>
                    <td>Backups management</td>
                    <td data-label="Postgresus">✅ Multiple DBs</td>
                    <td data-label="WAL-G">❌ Single DB only</td>
                  </tr>
                  <tr>
                    <td>Support of other DBs</td>
                    <td data-label="Postgresus">
                      ✅ PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                    <td data-label="WAL-G">✅ PostgreSQL, MySQL, MS SQL</td>
                  </tr>
                  <tr>
                    <td>Interface</td>
                    <td data-label="Postgresus">Web UI</td>
                    <td data-label="WAL-G">Command-line only</td>
                  </tr>
                  <tr>
                    <td>Backup type</td>
                    <td data-label="Postgresus">Logical (pg_dump)</td>
                    <td data-label="WAL-G">Physical (WAL archiving)</td>
                  </tr>
                  <tr>
                    <td>Backup scheduling</td>
                    <td data-label="Postgresus">✅ Built-in scheduler</td>
                    <td data-label="WAL-G">Requires external (cron)</td>
                  </tr>
                  <tr>
                    <td>Recovery options</td>
                    <td data-label="Postgresus">
                      ❌ No PITR (restore to any hour or day)
                    </td>
                    <td data-label="WAL-G">
                      ✅ WAL-based PITR (second-precise)
                    </td>
                  </tr>
                  <tr>
                    <td>Incremental backups</td>
                    <td data-label="Postgresus">
                      Full backups with compression
                    </td>
                    <td data-label="WAL-G">
                      Delta backups (changed pages only)
                    </td>
                  </tr>
                  <tr>
                    <td>Team features</td>
                    <td data-label="Postgresus">
                      ✅ Workspaces, RBAC, audit logs
                    </td>
                    <td data-label="WAL-G">❌ OS-level permissions only</td>
                  </tr>
                  <tr>
                    <td>Notifications</td>
                    <td data-label="Postgresus">
                      ✅ Slack, Teams, Telegram, Email
                    </td>
                    <td data-label="WAL-G">❌ Requires custom scripting</td>
                  </tr>
                  <tr>
                    <td>Encryption</td>
                    <td data-label="Postgresus">Built-in AES-256-GCM</td>
                    <td data-label="WAL-G">GPG or libsodium</td>
                  </tr>
                  <tr>
                    <td>Learning curve</td>
                    <td data-label="Postgresus">Minimal</td>
                    <td data-label="WAL-G">CLI proficiency required</td>
                  </tr>
                  <tr>
                    <td>Installation</td>
                    <td data-label="Postgresus">One-line script or Docker</td>
                    <td data-label="WAL-G">Binary download + configuration</td>
                  </tr>
                  <tr>
                    <td>Suitable for self-hosted DBs</td>
                    <td data-label="Postgresus">✅ Yes</td>
                    <td data-label="WAL-G">✅ Yes</td>
                  </tr>
                  <tr>
                    <td>Suitable for cloud DBs</td>
                    <td data-label="Postgresus">
                      ✅ Yes (RDS, Cloud SQL, Azure)
                    </td>
                    <td data-label="WAL-G">
                      ❌ Backup only (no restore to cloud)
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="database-focus">Database focus</h2>

              <p>
                One of the most significant differences between these tools is
                their database scope:
              </p>

              <h3 id="focus-postgresus">
                Postgresus: Comprehensive backup management
              </h3>

              <p>
                Postgresus is built for comprehensive backup management across
                multiple database systems with a focus on ease of use:
              </p>

              <ul>
                <li>
                  <strong>Multi-database support</strong>: Manage backups for
                  PostgreSQL, MySQL, MariaDB and MongoDB from a single
                  interface.
                </li>
                <li>
                  <strong>Unified experience</strong>: The interface, workflows
                  and features work consistently across all supported databases.
                </li>
                <li>
                  <strong>Version support</strong>: Supports PostgreSQL versions
                  12 through 18, with version-specific optimizations.
                </li>
                <li>
                  <strong>Streamlined management</strong>: All development
                  effort goes into improving the backup management experience.
                </li>
              </ul>

              <h3 id="focus-wal-g">WAL-G: Multi-database support</h3>

              <p>
                WAL-G started as a PostgreSQL backup tool but has expanded to
                support multiple database systems:
              </p>

              <ul>
                <li>
                  <strong>PostgreSQL</strong>: The original and most mature
                  implementation.
                </li>
                <li>
                  <strong>MySQL/MariaDB</strong>: Supports binlog-based backups.
                </li>
                <li>
                  <strong>MS SQL Server</strong>: Windows-based SQL Server
                  backups.
                </li>
                <li>
                  <strong>MongoDB</strong>: Document database backup support.
                </li>
                <li>
                  <strong>FoundationDB</strong>: Distributed database support.
                </li>
                <li>
                  <strong>Greenplum</strong>: Data warehouse backup support.
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">
                    When comprehensive management matters:
                  </strong>{" "}
                  If you need to manage backups for multiple databases with a
                  unified interface, Postgresus offers a streamlined experience.
                  You get centralized backup management without the complexity
                  of juggling different tools for different databases and with
                  team features.
                </p>
              </div>

              <h2 id="target-audience">Target audience</h2>

              <p>
                The tools serve different user profiles based on their design
                philosophy:
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
                <li>
                  <strong>Multi-database environments</strong>: Organizations
                  running PostgreSQL, MySQL, MariaDB or MongoDB benefit from
                  centralized backup management.
                </li>
              </ul>

              <h3 id="audience-wal-g">WAL-G audience</h3>

              <p>
                WAL-G is designed for users comfortable with command-line tools:
              </p>

              <ul>
                <li>
                  <strong>DevOps engineers</strong>: Those who prefer
                  infrastructure-as-code and CLI-based workflows.
                </li>
                <li>
                  <strong>Multi-database environments</strong>: Organizations
                  running PostgreSQL alongside MySQL, MongoDB or other supported
                  databases.
                </li>
                <li>
                  <strong>Cloud-native deployments</strong>: Teams using
                  Kubernetes or containerized environments where CLI tools
                  integrate well.
                </li>
                <li>
                  <strong>Users needing PITR</strong>: Those requiring
                  second-precise Point-in-Time Recovery for mission-critical
                  systems.
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

              <h3 id="backup-wal-g">
                WAL-G: Physical backups with WAL archiving
              </h3>

              <p>
                WAL-G performs file-level (physical) backups with continuous WAL
                archiving:
              </p>

              <ul>
                <li>
                  <strong>Base backups</strong>: Full file-level copies of the
                  PostgreSQL data directory.
                </li>
                <li>
                  <strong>Delta backups</strong>: Only changed pages are backed
                  up, reducing storage and transfer time.
                </li>
                <li>
                  <strong>WAL archiving</strong>: Continuous archiving of
                  Write-Ahead Logs enables Point-in-Time Recovery.
                </li>
                <li>
                  <strong>Copy-on-write optimization</strong>: Efficient
                  handling of unchanged data blocks.
                </li>
              </ul>

              <h2 id="recovery-options">Recovery options</h2>

              <p>
                Both tools offer recovery capabilities, but with different
                granularity:
              </p>

              <h3 id="recovery-postgresus">Postgresus recovery</h3>

              <ul>
                <li>
                  <strong>Restore to any hour or day</strong>: With hourly,
                  daily, weekly, monthly or cron backup schedules, you can
                  restore to any backup point you&apos;ve configured.
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

              <h3 id="recovery-wal-g">WAL-G recovery</h3>

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
                  <strong>Delta restore</strong>: Faster recovery by only
                  fetching changed pages.
                </li>
                <li>
                  <strong>Standby creation</strong>: Create PostgreSQL replicas
                  from backups for high availability setups.
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Note:</strong> For most
                  applications, restoring to the nearest hour or day (as
                  Postgresus provides) is sufficient. Second-precise PITR is
                  typically only required for mission-critical financial or
                  transactional systems where every transaction must be
                  recoverable.{" "}
                  <Link
                    href="/faq#why-no-pitr"
                    className="text-blue-400 hover:text-blue-300"
                  >
                    Learn why Postgresus doesn&apos;t support PITR →
                  </Link>
                </p>
              </div>

              <h2 id="ease-of-use">Ease of use</h2>

              <p>
                The tools differ significantly in their approach to user
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

              <h3 id="ease-wal-g">WAL-G user experience</h3>

              <ul>
                <li>
                  <strong>Command-line interface</strong>: All operations
                  performed via terminal commands like{" "}
                  <code>wal-g backup-push</code>,{" "}
                  <code>wal-g backup-fetch</code>.
                </li>
                <li>
                  <strong>Environment variables</strong>: Configuration
                  primarily through environment variables rather than config
                  files.
                </li>
                <li>
                  <strong>External scheduling</strong>: Requires cron jobs or
                  external orchestration for automated backups.
                </li>
                <li>
                  <strong>WAL archiving setup</strong>: Must configure
                  PostgreSQL&apos;s <code>archive_command</code> to integrate
                  with WAL-G.
                </li>
                <li>
                  <strong>CLI proficiency expected</strong>: Documentation
                  assumes familiarity with command-line tools and shell
                  scripting.
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

              <h3 id="team-wal-g">WAL-G team capabilities</h3>

              <p>
                WAL-G is a command-line tool without built-in team features:
              </p>

              <ul>
                <li>No user management or access control</li>
                <li>No audit logging of operations</li>
                <li>Team coordination requires external tools and processes</li>
                <li>
                  Access controlled via OS-level permissions and cloud IAM
                  policies
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

              <h3 id="security-wal-g">WAL-G security</h3>

              <ul>
                <li>
                  <strong>GPG encryption</strong>: Supports GPG-based encryption
                  for backup files.
                </li>
                <li>
                  <strong>libsodium encryption</strong>: Alternative encryption
                  using the libsodium library.
                </li>
                <li>
                  <strong>Cloud IAM integration</strong>: Leverages cloud
                  provider IAM for access control to storage.
                </li>
                <li>
                  <strong>No built-in credential management</strong>: Relies on
                  environment variables or external secret management.
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
                Both tools support cloud storage, with different focus areas:
              </p>

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

              <h3 id="storage-wal-g">WAL-G storage</h3>

              <p>Cloud-native storage options:</p>

              <ul>
                <li>Amazon S3</li>
                <li>Google Cloud Storage (GCS)</li>
                <li>Azure Blob Storage</li>
                <li>Swift (OpenStack)</li>
                <li>Local file system</li>
                <li>SSH/SFTP</li>
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

              <h3 id="notifications-wal-g">WAL-G notifications</h3>

              <p>
                WAL-G does not have built-in notification support. Notifications
                require:
              </p>

              <ul>
                <li>Custom scripting around backup commands</li>
                <li>External monitoring tools integration</li>
                <li>Manual log parsing and alerting setup</li>
                <li>
                  Integration with tools like Prometheus, Grafana or custom
                  solutions
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

              <h2 id="compression">Compression</h2>

              <p>Both tools offer compression to reduce backup sizes:</p>

              <h3 id="compression-postgresus">Postgresus compression</h3>

              <ul>
                <li>
                  <strong>zstd compression</strong>: Uses zstd at level 5 for
                  balanced speed and compression ratio.
                </li>
                <li>
                  <strong>4-8x size reduction</strong>: Typical compression
                  ratios with only ~20% runtime overhead.
                </li>
                <li>
                  <strong>Automatic</strong>: Compression is enabled by default
                  with no configuration needed.
                </li>
              </ul>

              <h3 id="compression-wal-g">WAL-G compression</h3>

              <ul>
                <li>
                  <strong>Multiple algorithms</strong>: Supports LZ4, LZMA,
                  Brotli and zstd.
                </li>
                <li>
                  <strong>Configurable levels</strong>: Fine-tune compression
                  ratio vs. speed tradeoffs.
                </li>
                <li>
                  <strong>Per-file compression</strong>: WAL files and base
                  backups can use different settings.
                </li>
              </ul>

              <h2 id="conclusion">Conclusion</h2>

              <p>
                Postgresus and WAL-G serve different needs in the PostgreSQL
                backup ecosystem. The right choice depends on your database
                environment, team structure and operational preferences.
              </p>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">
                    Choose Postgresus if:
                  </strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    You need comprehensive backup management for PostgreSQL from
                    a single interface
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
                    You want built-in scheduling without external cron setup
                  </li>
                  <li>
                    Restoring to any hour or day meets your recovery
                    requirements
                  </li>
                  <li>You want quick setup with minimal database expertise</li>
                  <li>Built-in backup encryption is important to you</li>
                  <li>
                    You use cloud-managed databases (AWS RDS, Google Cloud SQL,
                    Azure) or self-hosted databases
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-white m-0">
                  <strong>Choose WAL-G if:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    You manage multiple self-hosted database systems
                    (PostgreSQL, MySQL, MongoDB, etc.) and want a unified tool
                  </li>
                  <li>
                    You require second-precise Point-in-Time Recovery for
                    mission-critical self-hosted systems
                  </li>
                  <li>
                    Delta backups are important for reducing storage and
                    transfer time
                  </li>
                  <li>
                    You prefer command-line tools and infrastructure-as-code
                    workflows
                  </li>
                  <li>
                    You&apos;re comfortable setting up cron jobs and custom
                    notification scripts
                  </li>
                  <li>
                    Your team has DevOps expertise for CLI-based tool management
                  </li>
                  <li>
                    You&apos;re building a database platform and need to backup
                    customer databases with PITR capabilities
                  </li>
                </ul>
              </div>

              <p>
                For comprehensive backup management for PostgeSQL, Postgresus
                offers a more streamlined experience with its unified interface
                and built-in features — and works seamlessly with both
                self-hosted and cloud-managed databases. Also it is suitable for
                teams.
                <br />
                <br />
                WAL-G remains an excellent choice for organizations with
                self-hosted databases who prefer CLI-based workflows and need
                advanced features like delta backups and precise PITR, or for
                teams building database platforms that need to provide PITR
                capabilities to their customers.
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
