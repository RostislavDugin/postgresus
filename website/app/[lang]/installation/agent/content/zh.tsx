import type { Metadata } from "next";
import {
  OG_LOCALES,
  getLanguageAlternates,
  getLocalizedUrl,
} from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Agent 安装 - Databasus 文档",
  description:
    "安装 Databasus agent，为 PostgreSQL 数据库提供物理备份、增量备份、WAL 归档和时间点恢复 (PITR)。",
  keywords: [
    "Databasus agent",
    "PostgreSQL 物理备份",
    "WAL 归档",
    "PITR",
    "时间点恢复",
    "pg_basebackup",
    "增量备份",
    "灾难恢复",
    "PostgreSQL agent",
    "数据库备份 agent",
  ],
  openGraph: {
    title: "Agent 安装 - Databasus 文档",
    description:
      "安装 Databasus agent，为 PostgreSQL 数据库提供物理备份、增量备份、WAL 归档和时间点恢复 (PITR)。",
    type: "article",
    url: getLocalizedUrl("zh", "installation/agent"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "Agent 安装 - Databasus 文档",
    description:
      "安装 Databasus agent，为 PostgreSQL 数据库提供物理备份、增量备份、WAL 归档和时间点恢复 (PITR)。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "installation/agent"),
    languages: getLanguageAlternates("installation/agent"),
  },
  robots: "index, follow",
};

export default function AgentInstallationPage() {
  const downloadCommand = `curl -L -o databasus-agent "<DATABASUS_HOST>/api/v1/system/agent?arch=<ARCH>" && chmod +x databasus-agent`;

  const postgresqlConf = `wal_level = replica
archive_mode = on
archive_command = 'cp %p <WAL_QUEUE_DIR>/%f.tmp && mv <WAL_QUEUE_DIR>/%f.tmp <WAL_QUEUE_DIR>/%f'`;

  const postgresqlConfDocker = `wal_level = replica
archive_mode = on
archive_command = 'cp %p /wal-queue/%f.tmp && mv /wal-queue/%f.tmp /wal-queue/%f'`;

  const pgHbaEntry = `host    replication   all   127.0.0.1/32   md5`;

  const grantReplication = `ALTER ROLE <YOUR_PG_USER> WITH REPLICATION;`;

  const createWalDir = `mkdir -p /opt/databasus/wal-queue`;

  const walDirPermissions = `chown postgres:postgres /opt/databasus/wal-queue
chmod 755 /opt/databasus/wal-queue`;

  const dockerVolumeExample = `# In your docker run command:
docker run ... -v /opt/databasus/wal-queue:/wal-queue ...

# Or in docker-compose.yml:
volumes:
  - /opt/databasus/wal-queue:/wal-queue`;

  const dockerWalDirPermissions = `# Inside the container (or via docker exec):
chown postgres:postgres /wal-queue`;

  const startCommandHost = `./databasus-agent start \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --pg-host=localhost \\
  --pg-port=5432 \\
  --pg-user=<YOUR_PG_USER> \\
  --pg-password=<YOUR_PG_PASSWORD> \\
  --pg-type=host \\
  --pg-wal-dir=/opt/databasus/wal-queue`;

  const startCommandFolder = `./databasus-agent start \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --pg-host=localhost \\
  --pg-port=5432 \\
  --pg-user=<YOUR_PG_USER> \\
  --pg-password=<YOUR_PG_PASSWORD> \\
  --pg-type=host \\
  --pg-host-bin-dir=<PATH_TO_PG_BIN_DIR> \\
  --pg-wal-dir=/opt/databasus/wal-queue`;

  const startCommandDocker = `./databasus-agent start \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --pg-host=localhost \\
  --pg-port=5432 \\
  --pg-user=<YOUR_PG_USER> \\
  --pg-password=<YOUR_PG_PASSWORD> \\
  --pg-type=docker \\
  --pg-docker-container-name=<CONTAINER_NAME> \\
  --pg-wal-dir=/opt/databasus/wal-queue`;

  const restoreCommand = `./databasus-agent restore \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --backup-id=<BACKUP_ID> \\
  --target-dir=<PGDATA_DIR>`;

  const restoreCommandDocker = `./databasus-agent restore \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --backup-id=<BACKUP_ID> \\
  --pg-type=docker \\
  --target-dir=<HOST_PGDATA_PATH>`;

  const restoreCommandPitr = `./databasus-agent restore \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --backup-id=<BACKUP_ID> \\
  --target-dir=<PGDATA_DIR> \\
  --target-time=<RFC3339_TIMESTAMP>`;

  const archiveCommandCleanup = `# In <PGDATA_DIR>/postgresql.auto.conf, remove or comment out:
# archive_mode = on
# archive_command = '...'`;

  const dockerVolumeMountExample = `# PostgreSQL 17 and earlier
docker run -d -v <HOST_PGDATA_PATH>:/var/lib/postgresql/data postgres:17

# PostgreSQL 18+
docker run -d -v <HOST_PGDATA_PATH>:/var/lib/postgresql/18/docker postgres:18`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Agent 安装 - Databasus 文档",
            description:
              "安装 Databasus agent，为 PostgreSQL 数据库提供物理备份、增量备份、WAL 归档和时间点恢复 (PITR)。",
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

      <DocsNavbarComponent lang="zh" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="zh" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="agent-installation">Agent 模式</h1>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-red-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-red-500 mt-0.5 shrink-0"
                >
                  <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
                  <path d="M12 9v4M12 17h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <strong>Agent 备份已弃用。</strong> Databasus 现在使用
                    PostgreSQL 17 原生备份远程执行物理备份和 PITR
                    备份，数据库服务器上无需安装任何 agent。{" "}
                    <a
                      href="/zh/faq/#why-no-agent"
                      className="text-blue-400 hover:text-blue-300 underline"
                    >
                      了解原因以及现在 PITR 备份的工作方式
                    </a>
                    。
                  </p>
                </div>
              </div>

              <p className="text-lg text-gray-400">
                Databasus agent 为 PostgreSQL
                数据库提供物理备份、增量备份、WAL 归档和时间点恢复
                (PITR)。
              </p>

              {/* When to use */}
              <h2 id="when-to-use">什么时候使用 agent</h2>

              <p>
                对大多数数据库来说，
                <strong>远程备份是最简单的方式</strong>。Databasus
                通过网络直接连接数据库，用 pg_dump
                做逻辑备份，数据库服务器上不需要任何额外软件。远程备份对云托管数据库（RDS、Cloud
                SQL、Supabase）和自托管实例都适用。
              </p>

              <p>
                agent 是为远程备份不够用的场景设计的：
              </p>

              <ul>
                <li>
                  <strong>基于 PITR 的灾难恢复</strong>{" "}
                  &mdash; 恢复到两次备份之间的任意一秒，数据丢失接近于零
                </li>
                <li>
                  <strong>物理备份</strong> &mdash;
                  对整个数据库集群做文件级复制，大数据集的备份和恢复更快
                </li>
                <li>
                  <strong>不对外暴露的数据库</strong> &mdash; agent 主动向外连接
                  Databasus，数据库无需公开端点
                </li>
                <li>
                  <strong>增量备份</strong> &mdash; 持续的 WAL
                  段归档结合周期性的基础备份
                </li>
              </ul>

              {/* In-app guided setup */}
              <h2 id="in-app-setup">应用内引导式配置</h2>

              <p>
                Databasus 直接在界面里提供交互式的安装和恢复指引。打开某个数据库的
                agent 设置时，所有命令都已预填好你的具体值：架构、数据库
                ID、agent 令牌、Databasus 主机地址和 PostgreSQL
                部署类型。你可以逐条复制命令，在服务器上执行。
              </p>

              <p>
                下面的文档覆盖同样的步骤，供查阅参考，也方便更喜欢在界面之外按指南操作的用户。
              </p>

              {/* Requirements */}
              <h2 id="requirements">前提条件</h2>

              <ul>
                <li>
                  <strong>PostgreSQL 15 或更新版本</strong>
                </li>
                <li>
                  <strong>Linux</strong>（amd64 或 arm64）
                </li>
                <li>
                  <strong>网络访问</strong>：agent 能连到你的 Databasus
                  实例（仅出站连接，数据库不需要能被 Databasus 访问到）
                </li>
              </ul>

              {/* Installation */}
              <h2 id="installation">安装</h2>

              <h3 id="step-1-download">第 1 步 &mdash; 下载 agent</h3>

              <p>
                在运行 PostgreSQL 的服务器上下载 agent 二进制文件。把{" "}
                <code>&lt;DATABASUS_HOST&gt;</code> 替换成你的 Databasus
                实例地址，把 <code>&lt;ARCH&gt;</code> 替换成{" "}
                <code>amd64</code> 或 <code>arm64</code>。
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={downloadCommand} />
                </div>
              </div>

              <h3 id="step-2-postgresql-conf">
                第 2 步 &mdash; 配置 postgresql.conf
              </h3>

              <p>
                在 <code>postgresql.conf</code> 中添加或更新以下设置，然后
                <strong>重启 PostgreSQL</strong>。
              </p>

              <p>
                <strong>主机安装</strong>（把{" "}
                <code>&lt;WAL_QUEUE_DIR&gt;</code> 替换成实际路径，例如{" "}
                <code>/opt/databasus/wal-queue</code>）：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlConf}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={postgresqlConf} />
                </div>
              </div>

              <p>
                <strong>Docker 安装</strong>时，<code>archive_command</code>{" "}
                中的路径（<code>/wal-queue</code>）是
                <strong>容器内部</strong>的路径，必须和卷挂载的目标路径一致，见第
                5 步。
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlConfDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={postgresqlConfDocker} />
                </div>
              </div>

              <h3 id="step-3-pg-hba">第 3 步 &mdash; 配置 pg_hba.conf</h3>

              <p>
                在 <code>pg_hba.conf</code> 中加入下面这行。这是{" "}
                <code>pg_basebackup</code>{" "}
                做全量备份所必需的，不是为了流复制。按需调整地址和认证方式，然后重新加载
                PostgreSQL。
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{pgHbaEntry}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={pgHbaEntry} />
                </div>
              </div>

              <h3 id="step-4-replication">
                第 4 步 &mdash; 授予 replication 权限
              </h3>

              <p>
                这是运行 <code>pg_basebackup</code> 的 PostgreSQL
                要求，并不会创建副本。
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{grantReplication}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={grantReplication} />
                </div>
              </div>

              <h3 id="step-5-wal-queue">
                第 5 步 &mdash; 创建 WAL 队列目录
              </h3>

              <p>
                PostgreSQL 会把 WAL 归档文件放到这里，供 agent 上传。
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{createWalDir}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={createWalDir} />
                </div>
              </div>

              <p>
                确保该目录对 PostgreSQL 可写、对 agent 可读：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{walDirPermissions}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={walDirPermissions} />
                </div>
              </div>

              <p>
                <strong>Docker 安装</strong>时，WAL 队列目录必须是 PostgreSQL
                容器和主机之间共享的卷挂载。agent 从主机路径读取 WAL 文件，而
                PostgreSQL 通过 <code>archive_command</code>{" "}
                写入容器内的路径。
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerVolumeExample}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={dockerVolumeExample} />
                </div>
              </div>

              <p>
                确保容器内的目录属于 <code>postgres</code> 用户：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerWalDirPermissions}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={dockerWalDirPermissions} />
                </div>
              </div>

              <h3 id="step-6-start">第 6 步 &mdash; 启动 agent</h3>

              <p>
                把 <code>&lt;ANGLE_BRACKETS&gt;</code>{" "}
                占位符替换成你的实际值。
              </p>

              <p>
                <strong>系统级 PostgreSQL</strong>（pg_basebackup 已在 PATH
                中）：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={startCommandHost} />
                </div>
              </div>

              <p>
                <strong>安装在特定目录的 PostgreSQL</strong>（例如{" "}
                <code>/usr/lib/postgresql/17/bin</code>）：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandFolder}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={startCommandFolder} />
                </div>
              </div>

              <p>
                <strong>Docker</strong>（使用<strong>容器内部</strong>的
                PostgreSQL 端口，通常是 5432，而不是映射到主机的端口）：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={startCommandDocker} />
                </div>
              </div>

              <h3 id="after-installation">安装之后</h3>

              <ul>
                <li>
                  执行 <code>start</code> 后 agent 在后台运行
                </li>
                <li>
                  查看状态：<code>./databasus-agent status</code>
                </li>
                <li>
                  查看日志：工作目录下的 <code>databasus.log</code>
                </li>
                <li>
                  停止 agent：<code>./databasus-agent stop</code>
                </li>
              </ul>

              {/* Restore */}
              <h2 id="restore">从 agent 备份恢复</h2>

              <p>
                把物理备份或增量备份恢复到目标目录。做时间点恢复时，加上{" "}
                <code>--target-time</code> 参数恢复到指定时刻。
              </p>

              <h3 id="restore-step-1">第 1 步 &mdash; 下载 agent</h3>

              <p>
                在要执行恢复的服务器上下载 agent 二进制文件（命令与安装的第 1
                步相同）。
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={downloadCommand} />
                </div>
              </div>

              <h3 id="restore-step-2">第 2 步 &mdash; 停止 PostgreSQL</h3>

              <p>
                恢复前必须停止 PostgreSQL。目标目录必须为空。
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>pg_ctl -D &lt;PGDATA_DIR&gt; stop</code>
                </pre>
              </div>

              <p>Docker 环境下：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker stop &lt;CONTAINER_NAME&gt;</code>
                </pre>
              </div>

              <h3 id="restore-step-3">第 3 步 &mdash; 执行恢复</h3>

              <p>
                把 <code>&lt;YOUR_AGENT_TOKEN&gt;</code> 替换成你的 agent
                令牌，把 <code>&lt;PGDATA_DIR&gt;</code> 替换成一个空的
                PostgreSQL 数据目录路径。
              </p>

              <p>
                <strong>主机安装：</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={restoreCommand} />
                </div>
              </div>

              <p>
                <strong>Docker 安装</strong>（
                <code>&lt;HOST_PGDATA_PATH&gt;</code>{" "}
                是主机上将被挂载为容器 pgdata 卷的路径）：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommandDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={restoreCommandDocker} />
                </div>
              </div>

              <p>
                在（重新）创建 postgres 容器时，把{" "}
                <code>&lt;HOST_PGDATA_PATH&gt;</code> 挂载到容器的 PGDATA
                路径。该路径取决于主版本号：PostgreSQL 18+ 使用{" "}
                <code>/var/lib/postgresql/&lt;major&gt;/docker</code>
                ；PostgreSQL 17 及更早版本使用{" "}
                <code>/var/lib/postgresql/data</code>。
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerVolumeMountExample}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={dockerVolumeMountExample} />
                </div>
              </div>

              <p>
                要做<strong>时间点恢复</strong>（PITR），加上{" "}
                <code>--target-time</code> 和一个 RFC 3339 时间戳（例如{" "}
                <code>2025-01-15T14:30:00Z</code>）：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommandPitr}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={restoreCommandPitr} />
                </div>
              </div>

              <h3 id="restore-step-4">
                第 4 步 &mdash; 处理 archive_command
              </h3>

              <p>
                恢复出来的备份包含原来的 <code>archive_command</code>{" "}
                配置。恢复完成后 PostgreSQL 归档 WAL
                文件会失败，除非你二选一：
              </p>

              <ul>
                <li>
                  <strong>重新接入 agent</strong> &mdash; 挂载 WAL
                  队列目录并在恢复出来的实例上启动 Databasus
                  agent，与最初的配置相同。
                </li>
                <li>
                  <strong>关闭归档</strong> &mdash;
                  如果暂时不需要持续备份，在{" "}
                  <code>postgresql.auto.conf</code>{" "}
                  中注释掉或重置归档设置：
                </li>
              </ul>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{archiveCommandCleanup}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={archiveCommandCleanup} />
                </div>
              </div>

              <h3 id="restore-step-5">第 5 步 &mdash; 启动 PostgreSQL</h3>

              <p>
                启动 PostgreSQL 开始 WAL 恢复。它会自动重放 WAL 段。
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>pg_ctl -D &lt;PGDATA_DIR&gt; start</code>
                </pre>
              </div>

              <p>Docker 环境下：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker start &lt;CONTAINER_NAME&gt;</code>
                </pre>
              </div>

              <h3 id="restore-step-6">第 6 步 &mdash; 清理</h3>

              <p>
                恢复完成后，删除 WAL 恢复目录：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>rm -rf &lt;PGDATA_DIR&gt;/databasus-wal-restore/</code>
                </pre>
              </div>

              {/* How it works */}
              <h2 id="how-it-works">工作原理</h2>

              <p>
                Databasus agent 是一个轻量的 Go
                二进制程序，同时运行两个并发进程：
              </p>

              <ul>
                <li>
                  <strong>WAL 流式上传</strong> &mdash; 大约每 10
                  秒从队列目录取出 WAL 段文件并上传到 Databasus
                </li>
                <li>
                  <strong>周期性基础备份</strong> &mdash; 按配置的计划运行{" "}
                  <code>pg_basebackup</code>
                  ，为数据库集群创建完整的物理备份
                </li>
              </ul>

              <p>
                恢复时，agent 下载基础备份和所有相关的 WAL 段，然后在{" "}
                <code>postgresql.auto.conf</code> 中配置{" "}
                <code>recovery.signal</code> 和{" "}
                <code>restore_command</code>。PostgreSQL 启动后重放 WAL
                段，直到到达目标恢复点。
              </p>

              <p>
                连接始终由 agent 主动发起（出站方向）。数据库服务器不需要接受来自
                Databasus 的入站连接，因此适合私有网络和有防火墙的环境。
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
