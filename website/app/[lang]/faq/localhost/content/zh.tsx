import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import { CopyButton } from "@/app/components/CopyButton";

export const metadata: Metadata = {
  title: "如何备份 localhost 上的数据库 | Databasus",
  description:
    "了解如何用 Databasus 备份运行在 localhost 上的 PostgreSQL 数据库。配置 Docker host 网络模式以备份本地数据库。",
  keywords: [
    "Databasus",
    "localhost 备份",
    "本地 PostgreSQL 备份",
    "备份本地数据库",
    "Docker host 网络",
    "PostgreSQL 备份",
    "数据库备份",
    "localhost 数据库",
  ],
  openGraph: {
    title: "如何备份 localhost 上的数据库 | Databasus",
    description:
      "了解如何用 Databasus 备份运行在 localhost 上的 PostgreSQL 数据库。配置 Docker host 网络模式以备份本地数据库。",
    type: "article",
    url: getLocalizedUrl("zh", "faq/localhost"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "如何备份 localhost 上的数据库 | Databasus",
    description:
      "了解如何用 Databasus 备份运行在 localhost 上的 PostgreSQL 数据库。配置 Docker host 网络模式以备份本地数据库。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "faq/localhost"),
    languages: getLanguageAlternates("faq/localhost"),
  },
  robots: "index, follow",
};

export default function LocalhostPage() {
  const dockerComposeHost = `services:
  databasus:
    container_name: databasus
    image: databasus/databasus:latest
    network_mode: host
    volumes:
      - ./databasus-data:/databasus-data
    restart: unless-stopped`;

  const dockerRunHost = `docker run -d \\
  --name databasus \\
  --network host \\
  -v ./databasus-data:/databasus-data \\
  --restart unless-stopped \\
  databasus/databasus:latest`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "如何用 Databasus 备份 localhost 上的数据库",
            description:
              "用 Databasus 备份运行在 localhost 上的 PostgreSQL 数据库的分步指南",
            step: [
              {
                "@type": "HowToStep",
                name: "配置 Docker host 网络模式",
                text: "修改 Docker 配置以使用 host 网络模式，让容器可以访问 localhost 上的服务。",
              },
              {
                "@type": "HowToStep",
                name: "使用 Docker Compose 或 docker run",
                text: "在 Docker Compose 中设置 network_mode: host，或在 docker run 中使用 --network host 参数。",
              },
              {
                "@type": "HowToStep",
                name: "连接 localhost 数据库",
                text: "在 Databasus 的备份配置中使用 127.0.0.1 或 localhost 作为数据库主机。",
              },
            ],
          }),
        }}
      />

      <DocsNavbarComponent lang="zh" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="zh" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="localhost-backup">如何备份 localhost 上的数据库</h1>

              <p className="text-lg text-gray-400">
                了解在使用 Docker 时，如何配置 Databasus 来备份运行在
                宿主机（localhost）上的 PostgreSQL 数据库。
              </p>

              <h2 id="the-problem">问题所在</h2>

              <p>
                如果 Databasus 运行在 Docker 里，而你想备份宿主机
                （localhost）上的数据库，就需要把 Docker 配置为{" "}
                <strong>host 网络模式</strong>。
              </p>

              <p>
                默认情况下，Docker 容器运行在隔离网络中，无法访问{" "}
                <code>localhost</code> 上的服务。host
                网络模式让容器与宿主机共享网络命名空间。
              </p>

              <h2 id="docker-compose-solution">Docker Compose 的解决方案</h2>

              <p>
                修改 <code>docker-compose.yml</code> 文件，使用{" "}
                <code>network_mode: host</code>：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerComposeHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={dockerComposeHost} />
                </div>
              </div>

              <h2 id="docker-run-solution">docker run 的解决方案</h2>

              <p>
                使用 <code>--network host</code> 参数：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerRunHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={dockerRunHost} />
                </div>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">💡 提示：</strong>使用
                  host 网络模式时，在 Databasus 的备份配置中可以用{" "}
                  <code className="bg-[#374151] text-gray-200">127.0.0.1</code>{" "}
                  或{" "}
                  <code className="bg-[#374151] text-gray-200">localhost</code>{" "}
                  作为数据库主机连接本地数据库。你也可以不做端口映射，
                  直接通过{" "}
                  <code className="bg-[#374151] text-gray-200">
                    http://localhost:4005
                  </code>{" "}
                  访问 Databasus 界面。
                </p>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">
                    ⚠️ Windows 和 macOS 用户请注意：
                  </strong>{" "}
                  <code className="bg-[#374151] text-red-400">host</code>{" "}
                  网络模式只在 Linux 上原生可用。在 Windows 和 macOS
                  上，Docker 运行在 Linux 虚拟机中，因此备份配置里的
                  数据库主机地址应使用{" "}
                  <code className="bg-[#374151] text-gray-200">
                    host.docker.internal
                  </code>{" "}
                  而不是{" "}
                  <code className="bg-[#374151] text-gray-200">localhost</code>
                  。
                </p>
              </div>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/zh/faq"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← 返回常见问题
                </a>
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
