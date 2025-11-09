import type { Metadata } from "next";
import { CopyButton } from "../components/CopyButton";
import DocsNavbarComponent from "../components/DocsNavbarComponent";
import DocsSidebarComponent from "../components/DocsSidebarComponent";
import DocTableOfContentComponent from "../components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Installation - Postgresus Documentation",
  description:
    "Learn how to install Postgresus using automated script, Docker run or Docker Compose. Simple zero-config installation for your self-hosted PostgreSQL backup system.",
  keywords: [
    "Postgresus installation",
    "Docker installation",
    "PostgreSQL backup setup",
    "self-hosted backup",
    "Docker Compose",
    "database backup installation",
    "pg_dump setup",
  ],
  openGraph: {
    title: "Installation - Postgresus Documentation",
    description:
      "Learn how to install Postgresus using automated script, Docker run or Docker Compose. Simple zero-config installation for your self-hosted PostgreSQL backup system.",
    type: "article",
    url: "https://postgresus.com/installation",
  },
  twitter: {
    card: "summary",
    title: "Installation - Postgresus Documentation",
    description:
      "Learn how to install Postgresus using automated script, Docker run or Docker Compose. Simple zero-config installation for your self-hosted PostgreSQL backup system.",
  },
  alternates: {
    canonical: "https://postgresus.com/installation",
  },
  robots: "index, follow",
};

export default function InstallationPage() {
  const installScript = `sudo apt-get install -y curl && \\
sudo curl -sSL https://raw.githubusercontent.com/RostislavDugin/postgresus/refs/heads/main/install-postgresus.sh | sudo bash`;

  const dockerRun = `docker run -d \\
  --name postgresus \\
  -p 4005:4005 \\
  -v ./postgresus-data:/postgresus-data \\
  --restart unless-stopped \\
  rostislavdugin/postgresus:latest`;

  const dockerCompose = `services:
  postgresus:
    container_name: postgresus
    image: rostislavdugin/postgresus:latest
    ports:
      - "4005:4005"
    volumes:
      - ./postgresus-data:/postgresus-data
    restart: unless-stopped`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Installation - Postgresus Documentation",
            description:
              "Learn how to install Postgresus using automated script, Docker run or Docker Compose. Simple zero-config installation for your self-hosted PostgreSQL backup system.",
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
            datePublished: "2025-01-01",
            dateModified: "2025-01-01",
          }),
        }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "How to install Postgresus",
            description:
              "Step-by-step guide to install Postgresus PostgreSQL backup tool",
            step: [
              {
                "@type": "HowToStep",
                name: "Automated installation script",
                text: "Run the automated installation script to install Docker and set up Postgresus with automatic startup configuration.",
                itemListElement: [
                  {
                    "@type": "HowToDirection",
                    text: "Execute the curl command to download and run the installation script",
                  },
                ],
              },
              {
                "@type": "HowToStep",
                name: "Docker Run",
                text: "Use Docker run command to quickly start Postgresus container with data persistence.",
              },
              {
                "@type": "HowToStep",
                name: "Docker Compose",
                text: "Create a docker-compose.yml file and use Docker Compose for managed deployment.",
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
              <h1 id="installation">Installation</h1>

              <p className="text-lg text-gray-700">
                You have three ways to install Postgresus: automated script
                (recommended), simple Docker run or Docker Compose setup.
              </p>

              <h2 id="system-requirements">System requirements</h2>

              <p>
                Postgresus requires the following minimum system resources to
                run properly:
              </p>

              <ul>
                <li>
                  <strong>CPU</strong>: At least 1 CPU cores
                </li>
                <li>
                  <strong>RAM</strong>: Minimum 500 MB RAM
                </li>
                <li>
                  <strong>Storage</strong>: 5 GB for installation and as much as
                  you need for backups
                </li>
                <li>
                  <strong>Docker</strong>: Docker Engine 20.10+ and Docker
                  Compose v2.0+
                </li>
              </ul>

              <h2 id="option-1-automated-script">
                Option 1: installation script (recommended, Linux only)
              </h2>

              <p>The installation script will:</p>

              <ul>
                <li>
                  ✅ Install Docker with Docker Compose (if not already
                  installed)
                </li>
                <li>✅ Set up Postgresus</li>
                <li>✅ Configure automatic startup on system reboot</li>
              </ul>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{installScript}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={installScript} />
                </div>
              </div>

              <p>
                In this case Postgresus will be installed in{" "}
                <code>/opt/postgresus</code> directory.
              </p>

              <h2 id="option-2-docker-run">Option 2: Simple Docker run</h2>

              <p>The easiest way to run Postgresus:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerRun}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerRun} />
                </div>
              </div>

              <p>This single command will:</p>

              <ul>
                <li>✅ Start Postgresus</li>
                <li>
                  ✅ Store all data in <code>./postgresus-data</code> directory
                </li>
                <li>✅ Automatically restart on system reboot</li>
              </ul>

              <h2 id="option-3-docker-compose">
                Option 3: Docker Compose setup
              </h2>

              <p>
                Create a <code>docker-compose.yml</code> file with the following
                configuration:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerCompose}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerCompose} />
                </div>
              </div>

              <p>Then run:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker compose up -d</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text="docker compose up -d" />
                </div>
              </div>

              <p>Keep in mind that start up can take up to ~2 minutes.</p>

              <h2 id="getting-started">Getting started</h2>

              <p>After installation:</p>

              <ol>
                <li>
                  <strong>Launch and access Postgresus</strong>: Start
                  Postgresus and navigate to <code>http://localhost:4005</code>
                </li>
                <li>
                  <strong>Create your first backup job</strong>: Click &quot;New
                  Backup&quot; and configure your PostgreSQL database connection
                </li>
                <li>
                  <strong>Configure schedule</strong>: Set up your backup
                  schedule (hourly, daily, weekly or monthly)
                </li>
                <li>
                  <strong>Choose storage destination</strong>: Select where to
                  store your backups (local, S3, Google Drive, etc.)
                </li>
                <li>
                  <strong>Set up notifications</strong>: Add notification
                  channels (Slack, Telegram, Discord) to get alerts about backup
                  status
                </li>
                <li>
                  <strong>Start backing up</strong>: Save your configuration and
                  watch your first backup run!
                </li>
              </ol>

              <h2 id="how-to-update">How to update Postgresus?</h2>

              <p>
                To update Postgresus, you need to stop it, clean up Docker cache
                and restart the container.
              </p>

              <ol>
                <li>
                  Go to the directory where Postgresus is installed (usually{" "}
                  <code>/opt/postgresus</code>)
                </li>
                <li>
                  Stop the container: <code>docker compose stop</code>
                </li>
                <li>
                  Clean up Docker cache: <code>docker system prune -a</code>
                </li>
                <li>
                  Restart the container: <code>docker compose up -d</code>
                </li>
              </ol>

              <p>
                It will get the latest version of Postgresus from the Docker Hub
                (if you have not fixed the version in the{" "}
                <code>docker-compose.yml</code> file).
              </p>

              <h2 id="troubleshooting">Troubleshooting</h2>

              <h3 id="container-wont-start">Container won&apos;t start</h3>

              <p>If the container fails to start, check the logs:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker logs postgresus</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text="docker logs postgresus" />
                </div>
              </div>

              <h3 id="port-already-in-use">Port already in use</h3>

              <p>
                If port 4005 is already in use, you can change it in your
                docker-compose.yml:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>
                    ports:
                    {"\n  "}- &quot;8080:4005&quot; # Change 8080 to any
                    available port
                  </code>
                </pre>
              </div>

              <h3 id="permission-denied">Permission denied errors</h3>

              <p>If you encounter permission issues with the data directory:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>
                    sudo chown -R $USER:$USER ./postgresus-data
                    {"\n"}
                    chmod -R 755 ./postgresus-data
                  </code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton
                    text={`sudo chown -R $USER:$USER ./postgresus-data\nchmod -R 755 ./postgresus-data`}
                  />
                </div>
              </div>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
