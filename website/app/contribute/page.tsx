import type { Metadata } from "next";
import DocsNavbarComponent from "../components/DocsNavbarComponent";
import DocsSidebarComponent from "../components/DocsSidebarComponent";
import DocTableOfContentComponent from "../components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Contribute to Databasus | Contribution Guide",
  description:
    "Learn how to contribute to Databasus through promotion, documentation, social sharing or code development. Add new storages, notifiers or fix bugs.",
  keywords: [
    "Databasus contribution",
    "open source contribution",
    "contribute to Databasus",
    "add storage",
    "add notifier",
    "PostgreSQL backup contribution",
    "developer guide",
  ],
  openGraph: {
    title: "Contribute to Databasus | Contribution Guide",
    description:
      "Learn how to contribute to Databasus through promotion, documentation, social sharing or code development.",
    type: "article",
    url: "https://databasus.com/contribute",
  },
  twitter: {
    card: "summary",
    title: "Contribute to Databasus | Contribution Guide",
    description:
      "Learn how to contribute to Databasus through promotion, documentation, social sharing or code development.",
  },
  alternates: {
    canonical: "https://databasus.com/contribute",
  },
  robots: "index, follow",
};

export default function ContributePage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Contribute to Databasus",
            description:
              "Learn how to contribute to Databasus through promotion, documentation, social sharing or code development",
            author: {
              "@type": "Organization",
              name: "Databasus",
            },
            publisher: {
              "@type": "Organization",
              name: "Databasus",
              logo: {
                "@type": "ImageObject",
                url: "https://databasus.com/logo.svg",
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
              <h1 id="contribute">Contribute</h1>

              <p className="text-lg text-gray-400">
                Contribution to Databasus is welcome! I would appreciate any
                help with the project.
              </p>

              <p>There are two ways to contribute:</p>

              <ol>
                <li>Via promotion, docs and social sharing</li>
                <li>Via code and development</li>
              </ol>

              <h2 id="how-to-start-contributing">How to start contributing</h2>

              <h3 id="contribution-via-promotion">
                Contribution via promotion, docs and social sharing
              </h3>

              <p>
                The most of users of Databasus are developers, DevOps engineers
                or DBAs. Meaning we are technical people. As well as I -
                developer of Databasus.
              </p>

              <p>
                I know how to develop and build IT projects. But I&apos;m not a
                marketing expert. It&apos;s really hard for me to promote the
                project and get more users. That is vital for Databasus.
              </p>

              <p>So I would appreciate any help with:</p>

              <ul>
                <li>
                  writing tutorials and guides to Medium, Dev.to, Hackernoon and
                  other platforms
                </li>
                <li>writing articles and blog posts about Databasus</li>
                <li>creating videos and tutorials</li>
                <li>
                  posting social media posts on LinkedIn, X, Telegram, etc.
                </li>
                <li>
                  <strong>
                    even just sharing the project with your friends and
                    colleagues
                  </strong>{" "}
                  - it&apos;s already a big help!
                </li>
                <li>
                  share answers to questions about Databasus on Stack Overflow,
                  Reddit, etc.
                </li>
              </ul>

              <p>
                Feel free to share your publications with our community in{" "}
                <a
                  href="https://t.me/databasus_community"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Telegram
                </a>{" "}
                channel.
              </p>

              <h3 id="contribution-via-code">
                Contribution via code and development
              </h3>

              <p>If you are developer, you can mainly contribute to:</p>

              <ul>
                <li>
                  new storages (see{" "}
                  <a href="/contribute/how-to-add-storage">
                    how to add storage guide
                  </a>
                  )
                </li>
                <li>
                  new notifiers (see{" "}
                  <a href="/contribute/how-to-add-notifier">
                    how to add notifier guide
                  </a>
                  )
                </li>
                <li>bug fixing</li>
                <li>issues writing</li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">⚠️ Important:</strong> All
                  the major changes for more than 10-20 lines of code it is
                  better to discuss with me first. At least, inform me that you
                  take something into work. Message me in Telegram (
                  <a
                    href="https://t.me/rostislav_dugin"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 underline hover:text-blue-300"
                  >
                    @rostislav_dugin
                  </a>
                  ).
                </p>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-red-400">
                    🚫 Note about large PRs:
                  </strong>{" "}
                  If you submit a large PR, most likely it will be refused.
                  Because now there are a lot of AI PRs which take a huge amount
                  of time to review and merge. I will not be able to review
                  them.
                </p>
              </div>

              <h2 id="prerequisites">Prerequisites</h2>

              <ol>
                <li>
                  Read docs in <code>/docs</code> folder, <code>README.md</code>{" "}
                  in <code>/backend</code> and <code>/frontend</code> folders
                </li>
                <li>
                  Run both backend and frontend following the instructions in
                  their respective README.md files (for development)
                </li>
                <li>Read this file till the end</li>
              </ol>

              <h2 id="how-to-create-pr">How to create a pull request?</h2>

              <p>We use gitflow approach.</p>

              <ol>
                <li>Create a new branch from main</li>
                <li>Make changes</li>
                <li>Create a pull request to main</li>
                <li>Wait for review</li>
                <li>Merge pull request</li>
              </ol>

              <h3 id="commit-naming">Commit naming convention</h3>

              <p>
                Commits should be named in the following format depending on the
                type of change:
              </p>

              <ul>
                <li>
                  <code>FEATURE (area): What was done</code>
                </li>
                <li>
                  <code>FIX (area): What was fixed</code>
                </li>
                <li>
                  <code>REFACTOR (area): What was refactored</code>
                </li>
              </ul>

              <p>To see examples, look at commit history in main branch.</p>

              <h3 id="branch-naming">Branch naming convention</h3>

              <p>Branches should be named in the following format:</p>

              <ul>
                <li>
                  <code>feature/what_was_done</code>
                </li>
                <li>
                  <code>fix/what_was_fixed</code>
                </li>
                <li>
                  <code>refactor/what_was_refactored</code>
                </li>
              </ul>

              <p>Example:</p>

              <ul>
                <li>
                  <code>feature/add_support_of_kubernetes_helm</code>
                </li>
                <li>
                  <code>fix/make_healthcheck_optional</code>
                </li>
                <li>
                  <code>refactor/refactor_navbar</code>
                </li>
              </ul>

              <h3 id="before-any-pr">Before any PR, make sure:</h3>

              <ol>
                <li>You created critical tests for your changes</li>
                <li>
                  <code>make lint</code> is passing (for backend) and{" "}
                  <code>npm run lint</code> is passing (for frontend)
                </li>
                <li>All tests are passing</li>
                <li>Project is building successfully</li>
                <li>
                  All your commits should be squashed into one commit with
                  proper message (or to meaningful parts)
                </li>
                <li>Code do really refactored and production ready</li>
                <li>
                  You have one single PR per one feature (at least, if features
                  not connected) with explanation what was done and why with
                  comparison of alternatives
                </li>
                <li>
                  If there are UI changes,{" "}
                  <strong>
                    please record a demo screencast with short manual how it
                    works
                  </strong>
                </li>
              </ol>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
