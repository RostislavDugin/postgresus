import type { Metadata } from "next";
import DocsNavbarComponent from "../components/DocsNavbarComponent";
import DocsSidebarComponent from "../components/DocsSidebarComponent";
import DocTableOfContentComponent from "../components/DocTableOfContentComponent";
import { CopyButton } from "../components/CopyButton";

export const metadata: Metadata = {
  title: "Security - How Postgresus Protects Your Data | Postgresus",
  description:
    "Learn how Postgresus ensures enterprise-level security with AES-256-GCM encryption for sensitive data and backups, read-only database access, and comprehensive audit logging.",
  keywords: [
    "Postgresus security",
    "PostgreSQL backup security",
    "AES-256-GCM encryption",
    "database encryption",
    "backup encryption",
    "read-only database access",
    "enterprise security",
    "data protection",
    "secure backups",
  ],
  openGraph: {
    title: "Security - How Postgresus Protects Your Data | Postgresus",
    description:
      "Learn how Postgresus ensures enterprise-level security with AES-256-GCM encryption for sensitive data and backups, read-only database access, and comprehensive audit logging.",
    type: "article",
    url: "https://postgresus.com/security",
  },
  twitter: {
    card: "summary",
    title: "Security - How Postgresus Protects Your Data | Postgresus",
    description:
      "Learn how Postgresus ensures enterprise-level security with AES-256-GCM encryption for sensitive data and backups, read-only database access, and comprehensive audit logging.",
  },
  alternates: {
    canonical: "https://postgresus.com/security",
  },
  robots: "index, follow",
};

export default function SecurityPage() {
  const encryptionPipeline = `PostgreSQL pg_dump → Compression → Encryption → Cloud Storage`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Security - How Postgresus Protects Your Data",
            description:
              "Learn how Postgresus ensures enterprise-level security with AES-256-GCM encryption for sensitive data and backups, read-only database access, and comprehensive audit logging.",
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
              <h1 id="security">How Postgresus enforces security?</h1>

              <p className="text-lg text-gray-700">
                Postgresus is responsible for sensitive data:
              </p>

              <ul>
                <li>it accesses your DB;</li>
                <li>it backs it up (meaning makes a copy of data);</li>
                <li>
                  it keeps credentials to be able to access your DB on a regular
                  basis;
                </li>
                <li>
                  it saves backups in cloud storages (if you enable it) - so
                  other people can see them;
                </li>
              </ul>

              <p>
                Therefore,{" "}
                <strong>
                  there is a main priority for Postgresus to be enterprise-level
                  secure and reliable
                </strong>
                .
              </p>

              <p>To make sure:</p>

              <ul>
                <li>sensitive data is never exposed and always encrypted;</li>
                <li>
                  backups are encrypted and useless even if someone sees them in
                  the cloud storage;
                </li>
                <li>
                  Postgresus doesn&apos;t even receive access to DB with write
                  or update access;
                </li>
                <li>all actions are logged and can be audited;</li>
              </ul>

              <p>
                All these steps protect your data. As you know, there is no 100%
                secure system, but we do our best to make it as secure as
                possible. Even in case of hacking, nobody will be able to
                corrupt your data.
              </p>

              <p>Postgresus enforces security on three levels:</p>

              <ol>
                <li>Sensitive data encryption;</li>
                <li>Backups encryption;</li>
                <li>Read-only access to DB.</li>
              </ol>

              <h2 id="level-1-sensitive-data-encryption">
                Level 1: sensitive data encryption
              </h2>

              <p>
                Internally, Postgresus uses PostgreSQL DB to store connection
                details, configs, settings of notifiers and storages (S3, Google
                Drive, Dropbox, etc.).
              </p>

              <p>Any sensitive data is encrypted. For example:</p>

              <ul>
                <li>passwords</li>
                <li>tokens</li>
                <li>webhooks with secrets</li>
              </ul>

              <p>
                So in DB Postgresus keeps only hashes or encoded values. For
                encryption is used <strong>AES-256-GCM</strong> algorithm. Also,
                despite the encryption, those values are never exposed via API
                or UI.
              </p>

              <p>
                The secret key used for encryption is stored on local storage (
                <code>./postgresus-data/secret.key</code> by default) and is not
                present in the DB itself. So DB compromise doesn&apos;t give
                access to sensitive data.
              </p>

              <h2 id="level-2-backups-encryption">
                Level 2: backups encryption
              </h2>

              <p>
                Each backup file is encrypted on the fly during backup creation.
                Postgresus uses <strong>AES-256-GCM</strong> encryption
                algorithm, which ensures that backup data cannot be read without
                the encryption key and any tampering is detected during
                decryption.
              </p>

              <p>Backups flow through this pipeline:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{encryptionPipeline}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={encryptionPipeline} />
                </div>
              </div>

              <p>
                Each backup gets its own unique encryption key derived from:
              </p>

              <ul>
                <li>
                  Master key (stored in{" "}
                  <code>./postgresus-data/secret.key</code>)
                </li>
                <li>Backup ID</li>
                <li>Random salt (unique per backup)</li>
              </ul>

              <p>
                <strong>Result</strong>: Even if someone gains access to your
                cloud storage (S3, Google Drive, etc.), they cannot read the
                backups without your master key.
              </p>

              <h2 id="level-3-read-only-access">
                Level 3: read-only access to DB
              </h2>

              <p>
                Postgresus enforces the principle of least privilege - it only
                needs read access to create backups, never write access. This
                protects your database from accidental or malicious data
                corruption through the backup tool.
              </p>

              <p>
                Before accepting database credentials, Postgresus performs
                checks across three levels:
              </p>

              <ol>
                <li>
                  <strong>Role-level</strong>: Verifies the user is NOT a
                  superuser and cannot create roles or databases
                </li>
                <li>
                  <strong>Database-level</strong>: Ensures no CREATE or TEMP
                  privileges
                </li>
                <li>
                  <strong>Table-level</strong>: Confirms zero write permissions
                  (INSERT, UPDATE, DELETE, TRUNCATE, etc.)
                </li>
              </ol>

              <p>
                The database user must pass all three checks to be considered
                read-only. If any write privilege is detected, Postgresus will
                warn you.
              </p>

              <p>
                Postgresus suggests creating read-only users for you with proper
                permissions:
              </p>

              <ul>
                <li>Grants SELECT on all current and future tables</li>
                <li>Grants USAGE on schemas (but not CREATE)</li>
                <li>Explicitly revokes all write privileges</li>
              </ul>

              <p>
                <strong>Result</strong>: Even if Postgresus is compromised,
                server is hacked, secret key is stolen and credentials are
                decrypted, attackers cannot corrupt your database.
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
