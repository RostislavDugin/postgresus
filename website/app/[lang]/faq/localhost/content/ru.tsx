import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import { CopyButton } from "@/app/components/CopyButton";

export const metadata: Metadata = {
  title: "Как делать бекапы баз данных на localhost | Databasus",
  description:
    "Как настроить резервное копирование PostgreSQL на localhost с помощью Databasus. Настройка режима host network в Docker для бекапов локальных баз данных.",
  keywords: [
    "Databasus",
    "бекап localhost",
    "бекап локального PostgreSQL",
    "резервное копирование локальной базы данных",
    "Docker host network",
    "резервное копирование PostgreSQL",
    "бекап базы данных",
    "база данных на localhost",
  ],
  openGraph: {
    title: "Как делать бекапы баз данных на localhost | Databasus",
    description:
      "Как настроить резервное копирование PostgreSQL на localhost с помощью Databasus. Настройка режима host network в Docker для бекапов локальных баз данных.",
    type: "article",
    url: getLocalizedUrl("ru", "faq/localhost"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title: "Как делать бекапы баз данных на localhost | Databasus",
    description:
      "Как настроить резервное копирование PostgreSQL на localhost с помощью Databasus. Настройка режима host network в Docker для бекапов локальных баз данных.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "faq/localhost"),
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
            name: "Как делать бекапы баз данных на localhost с Databasus",
            description:
              "Пошаговое руководство по резервному копированию баз данных PostgreSQL на localhost с помощью Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Настройте режим host network в Docker",
                text: "Обновите конфигурацию Docker, включив режим host network, чтобы контейнер имел доступ к сервисам на localhost.",
              },
              {
                "@type": "HowToStep",
                name: "Используйте Docker Compose или Docker run",
                text: "Добавьте network_mode: host в Docker Compose или флаг --network host для Docker run.",
              },
              {
                "@type": "HowToStep",
                name: "Подключитесь к базе данных на localhost",
                text: "Укажите 127.0.0.1 или localhost как хост базы данных в настройках резервного копирования Databasus.",
              },
            ],
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
              <h1 id="localhost-backup">
                Как делать бекапы баз данных на localhost
              </h1>

              <p className="text-lg text-gray-400">
                Как настроить Databasus в Docker для резервного копирования баз
                данных PostgreSQL, запущенных на хост-машине (localhost).
              </p>

              <h2 id="the-problem">Проблема</h2>

              <p>
                Если Databasus запущен в Docker и вы хотите делать бекапы баз
                данных на хост-машине (localhost), нужно включить в Docker{" "}
                <strong>режим host network</strong>.
              </p>

              <p>
                По умолчанию контейнеры Docker работают в изолированной сети и
                не имеют доступа к сервисам на <code>localhost</code>. В режиме
                host network контейнер использует сеть хоста напрямую.
              </p>

              <h2 id="docker-compose-solution">Решение для Docker Compose</h2>

              <p>
                Добавьте в файл <code>docker-compose.yml</code> строку{" "}
                <code>network_mode: host</code>:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerComposeHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerComposeHost} lang="ru" />
                </div>
              </div>

              <h2 id="docker-run-solution">Решение для Docker run</h2>

              <p>
                Используйте флаг <code>--network host</code>:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerRunHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerRunHost} lang="ru" />
                </div>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">💡 Примечание:</strong> в
                  режиме host network вы можете подключаться к локальной базе
                  данных, указав{" "}
                  <code className="bg-[#374151] text-gray-200">127.0.0.1</code>{" "}
                  или{" "}
                  <code className="bg-[#374151] text-gray-200">localhost</code>{" "}
                  как хост в настройках резервного копирования Databasus.
                  Интерфейс Databasus также будет доступен напрямую по адресу{" "}
                  <code className="bg-[#374151] text-gray-200">
                    http://localhost:4005
                  </code>{" "}
                  без проброса портов.
                </p>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6 pb-0">
                <p className="text-sm text-gray-300 m-0">
                  <strong className="text-amber-400">
                    ⚠️ Важно для пользователей Windows и macOS:
                  </strong>{" "}
                  режим сети{" "}
                  <code className="bg-[#374151] text-red-400">host</code>{" "}
                  нативно работает только на Linux. На Windows и macOS Docker
                  работает внутри виртуальной машины Linux, поэтому вместо{" "}
                  <code className="bg-[#374151] text-gray-200">localhost</code>{" "}
                  в качестве адреса базы данных в настройках резервного
                  копирования нужно указывать{" "}
                  <code className="bg-[#374151] text-gray-200">
                    host.docker.internal
                  </code>
                  .
                </p>
              </div>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/ru/faq"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Назад к FAQ
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
