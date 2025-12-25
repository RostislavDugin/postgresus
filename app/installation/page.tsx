import type { Metadata } from "next";
import { CopyButton } from "../components/CopyButton";
import DocsNavbarComponent from "../components/DocsNavbarComponent";
import DocsSidebarComponent from "../components/DocsSidebarComponent";
import DocTableOfContentComponent from "../components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Installation - Databasus Documentation",
  description:
    "Learn how to install Databasus using automated script, Docker run, Docker Compose or Helm for Kubernetes. Simple zero-config installation for your self-hosted PostgreSQL backup system.",
  keywords: [
    "Databasus installation",
    "Docker installation",
    "PostgreSQL backup setup",
    "self-hosted backup",
    "Docker Compose",
    "database backup installation",
    "pg_dump setup",
    "Kubernetes",
    "Helm chart",
    "K8s deployment",
  ],
  openGraph: {
    title: "Installation - Databasus Documentation",
    description:
      "Learn how to install Databasus using automated script, Docker run, Docker Compose or Helm for Kubernetes. Simple zero-config installation for your self-hosted PostgreSQL backup system.",
    type: "article",
    url: "https://databasus.com/installation",
  },
  twitter: {
    card: "summary",
    title: "Installation - Databasus Documentation",
    description:
      "Learn how to install Databasus using automated script, Docker run, Docker Compose or Helm for Kubernetes. Simple zero-config installation for your self-hosted PostgreSQL backup system.",
  },
  alternates: {
    canonical: "https://databasus.com/installation",
  },
  robots: "index, follow",
};

export default function InstallationPage() {
  const installScript = `sudo apt-get install -y curl && \\
sudo curl -sSL https://raw.githubusercontent.com/databasus/databasus/refs/heads/main/install-databasus.sh | sudo bash`;

  const dockerRun = `docker run -d \\
  --name databasus \\
  -p 4005:4005 \\
  -v ./databasus-data:/databasus-data \\
  --restart unless-stopped \\
  databasus/databasus:latest`;

  const dockerCompose = `services:
  databasus:
    container_name: databasus
    image: databasus/databasus:latest
    ports:
      - "4005:4005"
    volumes:
      - ./databasus-data:/databasus-data
    restart: unless-stopped`;

  const helmInstallClusterIP = `helm install databasus oci://ghcr.io/databasus/charts/databasus \\
  -n databasus --create-namespace`;

  const helmPortForward = `kubectl port-forward svc/databasus-service 4005:4005 -n databasus
# Access at http://localhost:4005`;

  const helmInstallLoadBalancer = `helm install databasus oci://ghcr.io/databasus/charts/databasus \\
  -n databasus --create-namespace \\
  --set service.type=LoadBalancer`;

  const helmGetSvc = `kubectl get svc databasus-service -n databasus
# Access at http://<EXTERNAL-IP>:4005`;

  const helmInstallIngress = `helm install databasus oci://ghcr.io/databasus/charts/databasus \\
  -n databasus --create-namespace \\
  --set ingress.enabled=true \\
  --set ingress.hosts[0].host=backup.example.com`;

  const helmUpgrade = `helm upgrade databasus oci://ghcr.io/databasus/charts/databasus -n databasus`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Installation - Databasus Documentation",
            description:
              "Learn how to install Databasus using automated script, Docker run, Docker Compose or Helm for Kubernetes. Simple zero-config installation for your self-hosted PostgreSQL backup system.",
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
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "How to install Databasus",
            description:
              "Step-by-step guide to install Databasus PostgreSQL backup tool",
            step: [
              {
                "@type": "HowToStep",
                name: "Automated installation script",
                text: "Run the automated installation script to install Docker and set up Databasus with automatic startup configuration.",
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
                text: "Use Docker run command to quickly start Databasus container with data persistence.",
              },
              {
                "@type": "HowToStep",
                name: "Docker Compose",
                text: "Create a docker-compose.yml file and use Docker Compose for managed deployment.",
              },
              {
                "@type": "HowToStep",
                name: "Kubernetes with Helm",
                text: "Use the official Helm chart to deploy Databasus on Kubernetes with StatefulSet, persistent storage and optional ingress.",
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
              <h1 id="installation">Installation</h1>

              <p className="text-lg text-gray-400">
                You have four ways to install Databasus: automated script
                (recommended), simple Docker run, Docker Compose setup or Helm
                for Kubernetes.
              </p>

              <h2 id="system-requirements">System requirements</h2>

              <p>
                Databasus requires the following minimum system resources to run
                properly:
              </p>

              <ul>
                <li>
                  <strong>CPU</strong>: At least 1 CPU core
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
                <li>✅ Set up Databasus</li>
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
                In this case Databasus will be installed in{" "}
                <code>/opt/databasus</code> directory.
              </p>

              <h2 id="option-2-docker-run">Option 2: Simple Docker run</h2>

              <p>The easiest way to run Databasus:</p>

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
                <li>✅ Start Databasus</li>
                <li>
                  ✅ Store all data in <code>./databasus-data</code> directory
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

              <h2 id="option-4-helm">Option 4: Kubernetes with Helm</h2>

              <p>
                For Kubernetes deployments, install directly from the OCI
                registry. Choose your preferred access method based on your
                environment.
              </p>

              <h3 id="helm-clusterip">
                With ClusterIP + port-forward (development)
              </h3>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallClusterIP}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={helmInstallClusterIP} />
                </div>
              </div>

              <p>Access via port-forward:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmPortForward}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={helmPortForward} />
                </div>
              </div>

              <h3 id="helm-loadbalancer">
                With LoadBalancer (cloud environments)
              </h3>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallLoadBalancer}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={helmInstallLoadBalancer} />
                </div>
              </div>

              <p>Get the external IP and access Databasus:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmGetSvc}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={helmGetSvc} />
                </div>
              </div>

              <h3 id="helm-ingress">With Ingress (domain-based access)</h3>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallIngress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={helmInstallIngress} />
                </div>
              </div>

              <p>
                For more options (NodePort, TLS, HTTPRoute for Gateway API), see
                the{" "}
                <a
                  href="https://github.com/databasus/databasus/tree/main/deploy/helm"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Helm chart documentation
                </a>
                .
              </p>

              <h2 id="getting-started">Getting started</h2>

              <p>After installation:</p>

              <ol>
                <li>
                  <strong>Launch and access Databasus</strong>: Start Databasus
                  and navigate to <code>http://localhost:4005</code>
                </li>
                <li>
                  <strong>Create your first backup job</strong>: Click &quot;New
                  Backup&quot; and configure your PostgreSQL database connection
                </li>
                <li>
                  <strong>Configure schedule</strong>: Set up your backup
                  schedule (hourly, daily, weekly, monthly or cron)
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

              <h2 id="how-to-update">How to update Databasus?</h2>

              <h3 id="update-docker">Update Docker installation</h3>

              <p>
                To update Databasus running via Docker, you need to stop it,
                clean up Docker cache and restart the container.
              </p>

              <ol>
                <li>
                  Go to the directory where Databasus is installed (usually{" "}
                  <code>/opt/databasus</code>)
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
                It will get the latest version of Databasus from the Docker Hub
                (if you have not fixed the version in the{" "}
                <code>docker-compose.yml</code> file).
              </p>

              <h3 id="update-helm">Update Helm installation</h3>

              <p>
                To update Databasus running on Kubernetes via Helm, use the
                upgrade command:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmUpgrade}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={helmUpgrade} />
                </div>
              </div>

              <p>
                If you have custom values, add <code>-f values.yaml</code> or
                use <code>--set</code> flags to preserve your configuration.
                Helm will perform a rolling update to the new version.
              </p>

              <h2 id="postgresus-migration">Migrating from Postgresus</h2>

              <p>
                Databasus is the new name for Postgresus. If you&apos;re
                currently using Postgresus, you can continue using it or migrate
                to Databasus.
              </p>

              <p>
                <strong>Important:</strong> Simply renaming the Docker image
                isn&apos;t enough, as Postgresus and Databasus use different
                data folders and internal database naming.
              </p>

              <p>To migrate:</p>

              <ol>
                <li>
                  Stop your Postgresus container:{" "}
                  <code>docker compose stop</code>
                </li>
                <li>
                  Install Databasus using any of the methods above (use a
                  different volume path <code>./databasus-data</code>)
                </li>
                <li>
                  Manually recreate your databases, storages and notifiers in
                  Databasus
                </li>
              </ol>

              <p>
                You can run both Postgresus and Databasus side by side during
                migration by using different ports and volume paths.
              </p>

              <h2 id="troubleshooting">Troubleshooting</h2>

              <h3 id="container-wont-start">Container won&apos;t start</h3>

              <p>If the container fails to start, check the logs:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker logs databasus</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text="docker logs databasus" />
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
                    sudo chown -R $USER:$USER ./databasus-data
                    {"\n"}
                    chmod -R 755 ./databasus-data
                  </code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton
                    text={`sudo chown -R $USER:$USER ./databasus-data\nchmod -R 755 ./databasus-data`}
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
