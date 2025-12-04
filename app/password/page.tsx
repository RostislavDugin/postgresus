import type { Metadata } from "next";
import { CopyButton } from "../components/CopyButton";
import DocsNavbarComponent from "../components/DocsNavbarComponent";
import DocsSidebarComponent from "../components/DocsSidebarComponent";
import DocTableOfContentComponent from "../components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Reset password - Postgresus Documentation",
  description:
    "Learn how to reset user passwords in Postgresus using the built-in command-line tool. Quick and secure password recovery for your PostgreSQL backup system.",
  keywords: [
    "Postgresus password reset",
    "reset user password",
    "PostgreSQL backup password",
    "Docker password recovery",
    "password recovery",
    "Postgresus authentication",
  ],
  openGraph: {
    title: "Reset Password - Postgresus Documentation",
    description:
      "Learn how to reset user passwords in Postgresus using the built-in command-line tool. Quick and secure password recovery for your PostgreSQL backup system.",
    type: "article",
    url: "https://postgresus.com/password",
  },
  twitter: {
    card: "summary",
    title: "Reset Password - Postgresus Documentation",
    description:
      "Learn how to reset user passwords in Postgresus using the built-in command-line tool. Quick and secure password recovery for your PostgreSQL backup system.",
  },
  alternates: {
    canonical: "https://postgresus.com/password",
  },
  robots: "index, follow",
};

export default function PasswordResetPage() {
  const resetPasswordCommand = `docker exec -it postgresus ./main --new-password="YourNewSecurePassword123" --email="admin"`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Reset Password - Postgresus Documentation",
            description:
              "Learn how to reset user passwords in Postgresus using the built-in command-line tool.",
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
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "How to reset Postgresus user password",
            description:
              "Step-by-step guide to reset user passwords in Postgresus",
            step: [
              {
                "@type": "HowToStep",
                name: "Run password reset command",
                text: "Execute the docker exec command with your new password and user email.",
                itemListElement: [
                  {
                    "@type": "HowToDirection",
                    text: "Use docker exec to run the password reset command inside the Postgresus container",
                  },
                ],
              },
              {
                "@type": "HowToStep",
                name: "Verify password change",
                text: "Log in to Postgresus with your new password to confirm the change was successful.",
              },
            ],
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
              <h1 id="reset-password">Reset user password</h1>

              <h2 id="reset-password-command">Reset password command</h2>

              <p>
                To reset a user&apos;s password, use the following command on
                the server where Postgresus is running:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{resetPasswordCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={resetPasswordCommand} />
                </div>
              </div>

              <h2 id="parameters">Parameters</h2>

              <p>The command accepts the following parameters:</p>

              <ul>
                <li>
                  <strong>--new-password</strong>: The new password. Make sure
                  it&apos;s secure and contains a mix of letters, numbers and
                  special characters.
                </li>
                <li>
                  <strong>--email</strong>: The email address of the user whose
                  password you want to reset (e.g., <code>admin</code>,{" "}
                  <code>user@example.com</code>).
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
