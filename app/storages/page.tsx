import type { Metadata } from "next";
import Link from "next/link";
import DocsNavbarComponent from "../components/DocsNavbarComponent";
import DocsSidebarComponent from "../components/DocsSidebarComponent";
import DocTableOfContentComponent from "../components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Storages - Postgresus Documentation",
  description:
    "List of supported storage destinations for Postgresus backups including local storage, S3, Cloudflare R2, Google Drive, NAS and Dropbox.",
  keywords: [
    "Postgresus storages",
    "backup storage",
    "S3 storage",
    "Google Drive backup",
    "Cloudflare R2",
    "NAS backup",
    "Dropbox backup",
    "local storage",
  ],
  openGraph: {
    title: "Storages - Postgresus Documentation",
    description:
      "List of supported storage destinations for Postgresus backups including local storage, S3, Cloudflare R2, Google Drive, NAS and Dropbox.",
    type: "article",
    url: "https://postgresus.com/storages",
  },
  twitter: {
    card: "summary",
    title: "Storages - Postgresus Documentation",
    description:
      "List of supported storage destinations for Postgresus backups including local storage, S3, Cloudflare R2, Google Drive, NAS and Dropbox.",
  },
  alternates: {
    canonical: "https://postgresus.com/storages",
  },
  robots: "index, follow",
};

export default function StoragesPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Storages - Postgresus Documentation",
            description:
              "List of supported storage destinations for Postgresus backups including local storage, S3, Cloudflare R2, Google Drive, NAS and Dropbox.",
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
              <h1 id="storages">Storages</h1>

              <p className="text-lg text-gray-700">
                Postgresus supports multiple storage destinations for your
                PostgreSQL backups. Choose where to store your backup files
                based on your infrastructure and requirements.
              </p>

              <h2 id="supported-storages">Supported storages</h2>

              <ul>
                <li>
                  <strong>Local Storage</strong> - Store backups directly on
                  your server or VPS
                </li>
                <li>
                  <strong>S3</strong> - Amazon S3 and S3-compatible storage
                  services
                </li>
                <li>
                  <Link
                    href="/storages/cloudflare-r2"
                    className="font-semibold text-blue-600 hover:text-blue-800"
                  >
                    Cloudflare R2
                  </Link>{" "}
                  - S3-compatible object storage from Cloudflare
                </li>
                <li>
                  <Link
                    href="/storages/google-drive"
                    className="font-semibold text-blue-600 hover:text-blue-800"
                  >
                    Google Drive
                  </Link>{" "}
                  - Cloud storage from Google
                </li>
                <li>
                  <strong>NAS</strong> - Network-attached storage devices
                </li>
              </ul>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
