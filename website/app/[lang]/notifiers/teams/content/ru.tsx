import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title:
    "Как настроить уведомления в Microsoft Teams для Databasus | Databasus",
  description:
    "Пошаговое руководство по настройке уведомлений в Microsoft Teams об итогах резервного копирования PostgreSQL в Databasus. Как создать вебхук Teams и настроить уведомления.",
  keywords: [
    "Databasus",
    "уведомления в Microsoft Teams",
    "резервное копирование PostgreSQL",
    "вебхук Teams",
    "оповещения о бекапах",
    "уведомления о базах данных",
  ],
  openGraph: {
    title:
      "Как настроить уведомления в Microsoft Teams для Databasus | Databasus",
    description:
      "Пошаговое руководство по настройке уведомлений в Microsoft Teams об итогах резервного копирования PostgreSQL в Databasus. Как создать вебхук Teams и настроить уведомления.",
    type: "article",
    url: getLocalizedUrl("ru", "notifiers/teams"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title:
      "Как настроить уведомления в Microsoft Teams для Databasus | Databasus",
    description:
      "Пошаговое руководство по настройке уведомлений в Microsoft Teams об итогах резервного копирования PostgreSQL в Databasus. Как создать вебхук Teams и настроить уведомления.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "notifiers/teams"),
    languages: getLanguageAlternates("notifiers/teams"),
  },
  robots: "index, follow",
};

export default function TeamsPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Как настроить уведомления в Microsoft Teams для Databasus",
            description:
              "Пошаговое руководство по настройке уведомлений в Microsoft Teams об итогах резервного копирования PostgreSQL в Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Откройте канал Teams",
                text: "Перейдите в канал Microsoft Teams, в который хотите получать уведомления.",
              },
              {
                "@type": "HowToStep",
                name: "Откройте рабочие процессы",
                text: "Откройте раздел Workflows в вашем канале Teams.",
              },
              {
                "@type": "HowToStep",
                name: "Создайте новый рабочий процесс",
                text: "Создайте новый рабочий процесс для входящих вебхуков.",
              },
              {
                "@type": "HowToStep",
                name: "Выберите шаблон вебхука",
                text: "Выберите шаблон входящего вебхука из доступных вариантов.",
              },
              {
                "@type": "HowToStep",
                name: "Настройте вебхук",
                text: "Задайте имя вебхука и канал.",
              },
              {
                "@type": "HowToStep",
                name: "Скопируйте URL вебхука",
                text: "Скопируйте URL вебхука, который сгенерировал Teams.",
              },
              {
                "@type": "HowToStep",
                name: "Настройте Databasus",
                text: "Вставьте URL вебхука в настройки уведомлений Databasus.",
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
              <h1 id="teams-notifications">Уведомления в Microsoft Teams</h1>

              <p className="text-lg text-gray-400">
                Настройте Microsoft Teams, чтобы мгновенно получать уведомления
                о статусе резервного копирования PostgreSQL. Оповещения об
                успешных бекапах, сбоях и предупреждениях будут приходить прямо
                в ваши каналы Teams.
              </p>

              <h2 id="setup-teams-webhook">Настройка вебхука Teams</h2>

              <h3 id="open-teams-channel">1. Откройте канал Teams</h3>

              <p>
                Перейдите в канал Microsoft Teams, в который хотите получать
                уведомления о бекапах. Нажмите на три точки (
                <strong>•••</strong>) рядом с названием канала.
              </p>

              <Image
                src="/images/notifier-teams/image-01.png"
                alt="Открытие канала Teams"
                width={800}
                height={500}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="access-workflows">2. Откройте рабочие процессы</h3>

              <p>
                В меню канала выберите <strong>&quot;Workflows&quot;</strong>,
                чтобы открыть интеграцию с Power Automate.
              </p>

              <Image
                src="/images/notifier-teams/image-02.png"
                alt="Открытие Workflows"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="create-new-workflow">
                3. Создайте новый рабочий процесс
              </h3>

              <p>
                В панели Workflows нажмите <strong>&quot;Create&quot;</strong>{" "}
                или найдите шаблон{" "}
                <strong>
                  &quot;Post to a channel when a webhook request is
                  received&quot;
                </strong>
                .
              </p>

              <Image
                src="/images/notifier-teams/image-03.png"
                alt="Создание нового рабочего процесса"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="select-webhook-template">4. Выберите шаблон вебхука</h3>

              <p>
                Выберите шаблон{" "}
                <strong>
                  &quot;Post to a channel when a webhook request is
                  received&quot;
                </strong>{" "}
                из доступных вариантов.
              </p>

              <Image
                src="/images/notifier-teams/image-04.png"
                alt="Выбор шаблона вебхука"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="configure-webhook">5. Настройте вебхук</h3>

              <p>
                Задайте имя вебхука (например,{" "}
                <strong>&quot;Databasus Backup Notifications&quot;</strong>) и
                подтвердите канал, в который будут приходить уведомления.
              </p>

              <Image
                src="/images/notifier-teams/image-05.png"
                alt="Настройка вебхука"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="copy-webhook-url">6. Скопируйте URL вебхука</h3>

              <p>
                После создания рабочего процесса вы увидите{" "}
                <strong>HTTP POST URL</strong>. Скопируйте этот URL: он
                понадобится при настройке Databasus.
              </p>

              <Image
                src="/images/notifier-teams/image-06.png"
                alt="Копирование URL вебхука"
                width={500}
                height={500}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="configure-databasus">Настройте Databasus</h2>

              <h3 id="add-teams-notifier">1. Добавьте Teams-уведомление</h3>

              <p>
                В Databasus откройте настройки уведомлений и добавьте новое
                уведомление Microsoft Teams. Вставьте URL вебхука, скопированный
                из Teams.
              </p>

              <Image
                src="/images/notifier-teams/image-07.png"
                alt="Настройка Teams в Databasus"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="test-notification">2. Проверьте уведомление</h3>

              <p>
                После настройки вебхука отправьте тестовое уведомление и
                убедитесь, что все работает. Тестовое сообщение должно прийти в
                выбранный канал Teams.
              </p>

              <p>
                Готово! Теперь ваш канал Microsoft Teams будет получать
                уведомления о бекапах PostgreSQL из Databasus.
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/ru/notifiers"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Назад к уведомлениям
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
