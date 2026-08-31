import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Установка агента - документация Databasus",
  description:
    "Установка агента Databasus для физических и инкрементальных бекапов, архивирования WAL и Point-in-Time Recovery (PITR) баз данных PostgreSQL.",
  keywords: [
    "агент Databasus",
    "физический бекап PostgreSQL",
    "архивирование WAL",
    "PITR",
    "Point-in-Time Recovery",
    "pg_basebackup",
    "инкрементальный бекап",
    "аварийное восстановление",
    "агент PostgreSQL",
    "агент резервного копирования баз данных",
  ],
  openGraph: {
    title: "Установка агента - документация Databasus",
    description:
      "Установка агента Databasus для физических и инкрементальных бекапов, архивирования WAL и Point-in-Time Recovery (PITR) баз данных PostgreSQL.",
    type: "article",
    url: getLocalizedUrl("ru", "installation/agent"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title: "Установка агента - документация Databasus",
    description:
      "Установка агента Databasus для физических и инкрементальных бекапов, архивирования WAL и Point-in-Time Recovery (PITR) баз данных PostgreSQL.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "installation/agent"),
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
            headline: "Установка агента - документация Databasus",
            description:
              "Установка агента Databasus для физических и инкрементальных бекапов, архивирования WAL и Point-in-Time Recovery (PITR) баз данных PostgreSQL.",
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

      <DocsNavbarComponent lang="ru" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="ru" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="agent-installation">Режим агента</h1>

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
                    <strong>Бекапы через агент устарели.</strong> Теперь
                    Databasus выполняет физические и PITR-бекапы удаленно, на
                    нативном механизме бекапов PostgreSQL 17, без установки
                    агента на сервер базы данных.{" "}
                    <a
                      href="/ru/faq/#why-no-agent"
                      className="text-blue-400 hover:text-blue-300 underline"
                    >
                      Почему так и как теперь работают PITR-бекапы
                    </a>
                    .
                  </p>
                </div>
              </div>

              <p className="text-lg text-gray-400">
                Агент Databasus умеет делать физические и инкрементальные
                бекапы, архивировать WAL и выполнять Point-in-Time Recovery
                (PITR) для баз данных PostgreSQL.
              </p>

              {/* When to use */}
              <h2 id="when-to-use">Когда нужен агент</h2>

              <p>
                Для большинства баз{" "}
                <strong>удаленные бекапы — самый простой вариант</strong>.
                Databasus подключается к базе напрямую по сети, делает
                логические бекапы через pg_dump и не требует дополнительного ПО
                на сервере базы данных. Удаленные бекапы работают и с облачными
                управляемыми базами (RDS, Cloud SQL, Supabase), и с
                self-hosted-инстансами.
              </p>

              <p>
                Агент рассчитан на случаи, когда удаленных бекапов недостаточно:
              </p>

              <ul>
                <li>
                  <strong>Аварийное восстановление с PITR</strong> —
                  восстановление на любую секунду между бекапами с почти нулевой
                  потерей данных
                </li>
                <li>
                  <strong>Физические бекапы</strong> — копия всего кластера базы
                  на уровне файлов: на больших объемах данных быстрее и бекап, и
                  восстановление
                </li>
                <li>
                  <strong>Базы, недоступные снаружи</strong> — агент сам
                  подключается к Databasus в исходящем направлении, поэтому базе
                  не нужен публичный адрес
                </li>
                <li>
                  <strong>Инкрементальные бекапы</strong> — непрерывное
                  архивирование WAL-сегментов в сочетании с периодическими
                  базовыми бекапами
                </li>
              </ul>

              {/* In-app guided setup */}
              <h2 id="in-app-setup">Пошаговая настройка в интерфейсе</h2>

              <p>
                Databasus показывает интерактивные инструкции по установке и
                восстановлению прямо в интерфейсе. Когда вы открываете настройки
                агента для базы, все команды уже заполнены вашими значениями:
                архитектурой, ID базы, токеном агента, адресом Databasus и типом
                развертывания PostgreSQL. Остается скопировать каждую команду и
                выполнить ее на своем сервере.
              </p>

              <p>
                Документация ниже описывает те же шаги — как справочник и для
                тех, кому удобнее идти по руководству вне интерфейса.
              </p>

              {/* Requirements */}
              <h2 id="requirements">Требования</h2>

              <ul>
                <li>
                  <strong>PostgreSQL 15 или новее</strong>
                </li>
                <li>
                  <strong>Linux</strong> (amd64 или arm64)
                </li>
                <li>
                  <strong>Сетевой доступ</strong> от агента к вашему инстансу
                  Databasus (только исходящий — базе не нужно быть доступной со
                  стороны Databasus)
                </li>
              </ul>

              {/* Installation */}
              <h2 id="installation">Установка</h2>

              <h3 id="step-1-download">Шаг 1 — скачайте агент</h3>

              <p>
                Скачайте бинарный файл агента на сервер, где работает
                PostgreSQL. Замените <code>&lt;DATABASUS_HOST&gt;</code> на URL
                вашего инстанса Databasus, а <code>&lt;ARCH&gt;</code> — на{" "}
                <code>amd64</code> или <code>arm64</code>.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={downloadCommand} />
                </div>
              </div>

              <h3 id="step-2-postgresql-conf">
                Шаг 2 — настройте postgresql.conf
              </h3>

              <p>
                Добавьте или обновите эти настройки в{" "}
                <code>postgresql.conf</code>, затем{" "}
                <strong>перезапустите PostgreSQL</strong>.
              </p>

              <p>
                <strong>Для установки на хосте</strong> (замените{" "}
                <code>&lt;WAL_QUEUE_DIR&gt;</code> на реальный путь, например{" "}
                <code>/opt/databasus/wal-queue</code>):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlConf}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={postgresqlConf} />
                </div>
              </div>

              <p>
                <strong>Для установки в Docker</strong> путь в{" "}
                <code>archive_command</code> (<code>/wal-queue</code>) — это
                путь <strong>внутри контейнера</strong>. Он должен совпадать с
                точкой монтирования тома — см. шаг 5.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlConfDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={postgresqlConfDocker} />
                </div>
              </div>

              <h3 id="step-3-pg-hba">Шаг 3 — настройте pg_hba.conf</h3>

              <p>
                Добавьте эту строку в <code>pg_hba.conf</code>. Она нужна, чтобы{" "}
                <code>pg_basebackup</code> мог снимать полные бекапы, а не для
                потоковой репликации. При необходимости поправьте адрес и метод
                аутентификации, затем перечитайте конфигурацию PostgreSQL.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{pgHbaEntry}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={pgHbaEntry} />
                </div>
              </div>

              <h3 id="step-4-replication">Шаг 4 — выдайте право репликации</h3>

              <p>
                Это требование PostgreSQL для запуска <code>pg_basebackup</code>{" "}
                — реплика при этом не создается.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{grantReplication}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={grantReplication} />
                </div>
              </div>

              <h3 id="step-5-wal-queue">
                Шаг 5 — создайте каталог очереди WAL
              </h3>

              <p>
                Сюда PostgreSQL складывает архивные WAL-файлы, а агент их
                загружает.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{createWalDir}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={createWalDir} />
                </div>
              </div>

              <p>
                Убедитесь, что PostgreSQL может писать в каталог, а агент —
                читать из него:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{walDirPermissions}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={walDirPermissions} />
                </div>
              </div>

              <p>
                <strong>Для установки в Docker</strong> каталог очереди WAL
                должен быть томом, общим для контейнера PostgreSQL и хоста.
                Агент читает WAL-файлы по пути на хосте, а PostgreSQL пишет по
                пути внутри контейнера через <code>archive_command</code>.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerVolumeExample}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={dockerVolumeExample} />
                </div>
              </div>

              <p>
                Убедитесь, что каталог внутри контейнера принадлежит
                пользователю <code>postgres</code>:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerWalDirPermissions}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={dockerWalDirPermissions} />
                </div>
              </div>

              <h3 id="step-6-start">Шаг 6 — запустите агент</h3>

              <p>
                Замените плейсхолдеры в <code>&lt;ANGLE_BRACKETS&gt;</code> на
                реальные значения.
              </p>

              <p>
                <strong>PostgreSQL, установленный в систему</strong>{" "}
                (pg_basebackup доступен в PATH):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={startCommandHost} />
                </div>
              </div>

              <p>
                <strong>PostgreSQL в отдельном каталоге</strong> (например{" "}
                <code>/usr/lib/postgresql/17/bin</code>):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandFolder}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={startCommandFolder} />
                </div>
              </div>

              <p>
                <strong>Docker</strong> (используйте порт PostgreSQL{" "}
                <strong>внутри контейнера</strong>, обычно 5432, а не порт,
                проброшенный на хост):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={startCommandDocker} />
                </div>
              </div>

              <h3 id="after-installation">После установки</h3>

              <ul>
                <li>
                  После команды <code>start</code> агент работает в фоне
                </li>
                <li>
                  Проверить статус: <code>./databasus-agent status</code>
                </li>
                <li>
                  Посмотреть логи: файл <code>databasus.log</code> в рабочем
                  каталоге
                </li>
                <li>
                  Остановить агент: <code>./databasus-agent stop</code>
                </li>
              </ul>

              {/* Restore */}
              <h2 id="restore">Восстановление из бекапа агента</h2>

              <p>
                Восстановите физический или инкрементальный бекап в целевой
                каталог. Для Point-in-Time Recovery добавьте флаг{" "}
                <code>--target-time</code>, чтобы восстановиться на конкретный
                момент.
              </p>

              <h3 id="restore-step-1">Шаг 1 — скачайте агент</h3>

              <p>
                Скачайте бинарный файл агента на сервер, где будете
                восстанавливать базу (та же команда, что и в шаге 1 установки).
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={downloadCommand} />
                </div>
              </div>

              <h3 id="restore-step-2">Шаг 2 — остановите PostgreSQL</h3>

              <p>
                Перед восстановлением PostgreSQL должен быть остановлен, а
                целевой каталог — пуст.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>pg_ctl -D &lt;PGDATA_DIR&gt; stop</code>
                </pre>
              </div>

              <p>Для Docker:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker stop &lt;CONTAINER_NAME&gt;</code>
                </pre>
              </div>

              <h3 id="restore-step-3">Шаг 3 — запустите восстановление</h3>

              <p>
                Замените <code>&lt;YOUR_AGENT_TOKEN&gt;</code> на токен агента,
                а <code>&lt;PGDATA_DIR&gt;</code> — на путь к пустому каталогу
                данных PostgreSQL.
              </p>

              <p>
                <strong>Установка на хосте:</strong>
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={restoreCommand} />
                </div>
              </div>

              <p>
                <strong>Установка в Docker</strong> (
                <code>&lt;HOST_PGDATA_PATH&gt;</code> — путь на хосте, который
                будет смонтирован как том pgdata контейнера):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommandDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={restoreCommandDocker} />
                </div>
              </div>

              <p>
                Смонтируйте <code>&lt;HOST_PGDATA_PATH&gt;</code> в путь PGDATA
                при (пере)создании контейнера postgres. Путь зависит от мажорной
                версии: PostgreSQL 18+ использует{" "}
                <code>/var/lib/postgresql/&lt;major&gt;/docker</code>;
                PostgreSQL 17 и старее — <code>/var/lib/postgresql/data</code>.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerVolumeMountExample}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={dockerVolumeMountExample} />
                </div>
              </div>

              <p>
                Для <strong>Point-in-Time Recovery</strong> (PITR) добавьте{" "}
                <code>--target-time</code> с меткой времени RFC 3339 (например{" "}
                <code>2025-01-15T14:30:00Z</code>):
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommandPitr}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={restoreCommandPitr} />
                </div>
              </div>

              <h3 id="restore-step-4">Шаг 4 — разберитесь с archive_command</h3>

              <p>
                Восстановленный бекап включает исходную конфигурацию{" "}
                <code>archive_command</code>. После восстановления PostgreSQL не
                сможет архивировать WAL-файлы, пока вы не сделаете одно из двух:
              </p>

              <ul>
                <li>
                  <strong>Снова подключите агент</strong> — смонтируйте каталог
                  очереди WAL и запустите агент Databasus на восстановленном
                  инстансе, так же как в исходной настройке.
                </li>
                <li>
                  <strong>Отключите архивирование</strong> — если непрерывные
                  бекапы пока не нужны, закомментируйте или сбросьте настройки
                  архивирования в <code>postgresql.auto.conf</code>:
                </li>
              </ul>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{archiveCommandCleanup}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={archiveCommandCleanup} />
                </div>
              </div>

              <h3 id="restore-step-5">Шаг 5 — запустите PostgreSQL</h3>

              <p>
                Запустите PostgreSQL, чтобы началось восстановление по WAL. Он
                автоматически проиграет WAL-сегменты.
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>pg_ctl -D &lt;PGDATA_DIR&gt; start</code>
                </pre>
              </div>

              <p>Для Docker:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker start &lt;CONTAINER_NAME&gt;</code>
                </pre>
              </div>

              <h3 id="restore-step-6">Шаг 6 — приберитесь</h3>

              <p>
                Когда восстановление завершится, удалите каталог восстановления
                WAL:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>rm -rf &lt;PGDATA_DIR&gt;/databasus-wal-restore/</code>
                </pre>
              </div>

              {/* How it works */}
              <h2 id="how-it-works">Как это работает</h2>

              <p>
                Агент Databasus — легкий бинарник на Go, который запускает два
                параллельных процесса:
              </p>

              <ul>
                <li>
                  <strong>Передача WAL</strong> — примерно каждые 10 секунд
                  забирает WAL-сегменты из каталога очереди и загружает их в
                  Databasus
                </li>
                <li>
                  <strong>Периодические базовые бекапы</strong> — запускает{" "}
                  <code>pg_basebackup</code> по настроенному расписанию,
                  создавая полные физические бекапы кластера
                </li>
              </ul>

              <p>
                При восстановлении агент скачивает базовый бекап и все нужные
                WAL-сегменты, затем настраивает <code>recovery.signal</code> и{" "}
                <code>restore_command</code> в <code>postgresql.auto.conf</code>
                . Когда PostgreSQL стартует, он проигрывает WAL-сегменты до
                целевой точки восстановления.
              </p>

              <p>
                Соединение с Databasus всегда инициирует агент (исходящее).
                Серверу базы данных не нужно принимать входящие соединения от
                Databasus, поэтому агент подходит для приватных сетей и
                окружений за файрволом.
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
