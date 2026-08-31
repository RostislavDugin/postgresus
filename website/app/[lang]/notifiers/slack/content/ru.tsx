import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Как настроить уведомления в Slack для Databasus | Databasus",
  description:
    "Пошаговое руководство по настройке уведомлений в Slack об итогах резервного копирования PostgreSQL в Databasus. Как создать Slack-бота и настроить уведомления.",
  keywords: [
    "Databasus",
    "уведомления в Slack",
    "резервное копирование PostgreSQL",
    "токен Slack-бота",
    "Slack API",
    "оповещения о бекапах",
    "уведомления о базах данных",
  ],
  openGraph: {
    title: "Как настроить уведомления в Slack для Databasus | Databasus",
    description:
      "Пошаговое руководство по настройке уведомлений в Slack об итогах резервного копирования PostgreSQL в Databasus. Как создать Slack-бота и настроить уведомления.",
    type: "article",
    url: getLocalizedUrl("ru", "notifiers/slack"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title: "Как настроить уведомления в Slack для Databasus | Databasus",
    description:
      "Пошаговое руководство по настройке уведомлений в Slack об итогах резервного копирования PostgreSQL в Databasus. Как создать Slack-бота и настроить уведомления.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "notifiers/slack"),
    languages: getLanguageAlternates("notifiers/slack"),
  },
  robots: "index, follow",
};

export default function SlackPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Как настроить уведомления в Slack для Databasus",
            description:
              "Пошаговое руководство по настройке уведомлений в Slack об итогах резервного копирования PostgreSQL в Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Откройте Slack API",
                text: "Перейдите на https://api.slack.com/apps и войдите в свое рабочее пространство Slack.",
              },
              {
                "@type": "HowToStep",
                name: "Создайте новое приложение",
                text: "Нажмите кнопку 'Create New App' и выберите 'From scratch'.",
              },
              {
                "@type": "HowToStep",
                name: "Настройте права бота",
                text: "Откройте OAuth & Permissions и добавьте нужные scope в Bot Token Scopes: chat:write, channels:join, im:write и groups:write.",
              },
              {
                "@type": "HowToStep",
                name: "Установите в рабочее пространство",
                text: "Установите приложение в рабочее пространство и авторизуйте его.",
              },
              {
                "@type": "HowToStep",
                name: "Скопируйте токен бота",
                text: "Скопируйте Bot User OAuth Token, который начинается с 'xoxb-'.",
              },
              {
                "@type": "HowToStep",
                name: "Получите ID канала",
                text: "Откройте нужный канал и найдите Channel ID в сведениях о канале.",
              },
              {
                "@type": "HowToStep",
                name: "Добавьте бота в приватный канал",
                text: "Если канал приватный, пригласите бота в канал, упомянув его.",
              },
              {
                "@type": "HowToStep",
                name: "Настройте Databasus",
                text: "В Databasus укажите Bot Token и Channel ID в настройках Slack-уведомлений.",
              },
              {
                "@type": "HowToStep",
                name: "Проверьте уведомление",
                text: "Отправьте тестовое уведомление и убедитесь, что все работает.",
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
              <h1 id="slack-notifications">Уведомления в Slack</h1>

              <p className="text-lg text-gray-400">
                Настройте Slack, чтобы мгновенно получать уведомления о статусе
                резервного копирования PostgreSQL. Оповещения об успешных
                бекапах, сбоях и предупреждениях будут приходить прямо в ваши
                каналы Slack.
              </p>

              <h2 id="create-slack-app">Создайте приложение Slack</h2>

              <h3 id="go-to-slack-api">1. Откройте Slack API</h3>

              <p>
                Перейдите на{" "}
                <a
                  href="https://api.slack.com/apps"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  https://api.slack.com/apps
                </a>{" "}
                и войдите в свое рабочее пространство Slack.
              </p>

              <h3 id="create-new-app">2. Создайте новое приложение</h3>

              <p>
                Нажмите кнопку <strong>&quot;Create New App&quot;</strong>.
              </p>

              <h3 id="choose-from-scratch">
                3. Выберите &quot;From scratch&quot;
              </h3>

              <p>
                В появившемся окне выберите вариант{" "}
                <strong>&quot;From scratch&quot;</strong>.
              </p>

              <h3 id="name-your-app">4. Назовите приложение</h3>

              <p>
                Введите имя приложения (например, &quot;Databasus
                Notifications&quot;) и выберите рабочее пространство, куда
                хотите его установить. Нажмите{" "}
                <strong>&quot;Create App&quot;</strong>.
              </p>

              <h2 id="configure-bot-permissions">Настройте права бота</h2>

              <h3 id="navigate-to-oauth">
                5. Перейдите в OAuth &amp; Permissions
              </h3>

              <p>
                В левой боковой панели нажмите{" "}
                <strong>&quot;OAuth &amp; Permissions&quot;</strong>.
              </p>

              <img
                src="/images/notifier-slack/image-1.png"
                alt="Переход в OAuth &amp; Permissions"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h3 id="add-bot-scopes">
                6. Добавьте Bot Token Scopes (обязательно)
              </h3>

              <p>
                Прокрутите вниз до секции <strong>&quot;Scopes&quot;</strong> и
                в блоке <strong>&quot;Bot Token Scopes&quot;</strong> нажмите{" "}
                <strong>&quot;Add an OAuth Scope&quot;</strong>.
              </p>

              <p>Добавьте все перечисленные обязательные scope:</p>

              <ul>
                <li>
                  <code>chat:write</code> &mdash; отправка сообщений в каналы
                </li>
                <li>
                  <code>channels:join</code> &mdash; автоматическое подключение
                  бота к публичным каналам
                </li>
                <li>
                  <code>im:write</code> &mdash; отправка личных сообщений
                  пользователям
                </li>
                <li>
                  <code>groups:write</code> &mdash; отправка сообщений в
                  приватные каналы
                </li>
                <li>
                  <code>channels:history</code> &mdash; чтение истории канала
                </li>
              </ul>

              <img
                src="/images/notifier-slack/image-2.png"
                alt="Добавление Bot Token Scopes"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h2 id="install-app">
                Установите приложение в рабочее пространство
              </h2>

              <h3 id="install-to-workspace">
                7. Установите в рабочее пространство
              </h3>

              <p>
                Прокрутите страницу{" "}
                <strong>&quot;OAuth &amp; Permissions&quot;</strong> вверх и
                нажмите <strong>&quot;Install to Workspace&quot;</strong>.
              </p>

              <img
                src="/images/notifier-slack/image-3.png"
                alt="Установка в рабочее пространство"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h3 id="authorize-app">8. Авторизуйте приложение</h3>

              <p>
                Просмотрите запрашиваемые права и нажмите{" "}
                <strong>&quot;Allow&quot;</strong>, чтобы авторизовать
                приложение.
              </p>

              <h3 id="copy-bot-token">9. Скопируйте Bot User OAuth Token</h3>

              <p>
                После установки вы увидите{" "}
                <strong>&quot;Bot User OAuth Token&quot;</strong>. Он начинается
                с <code>xoxb-</code>. Скопируйте этот токен: он понадобится при
                настройке Databasus.
              </p>

              <h2 id="get-channel-id">Получите ID канала</h2>

              <h3 id="open-channel">10. Откройте нужный канал</h3>

              <p>
                В рабочем пространстве Slack откройте канал, в который хотите
                получать уведомления о бекапах.
              </p>

              <h3 id="get-channel-info">11. Найдите ID канала</h3>

              <p>
                Нажмите на название канала вверху, затем прокрутите вниз в
                сведениях о канале. Внизу секции &quot;About&quot; вы найдете{" "}
                <strong>Channel ID</strong>. Он начинается с <code>C</code> (для
                публичных каналов) или <code>G</code> (для приватных).
              </p>

              <p>Скопируйте этот Channel ID.</p>

              <img
                src="/images/notifier-slack/image-4.png"
                alt="Получение ID канала"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[500px]"
                loading="lazy"
              />

              <h3 id="add-bot-to-channel">
                12. Добавьте бота в канал (обязательно для приватных каналов)
              </h3>

              <p>
                <strong>
                  Если вы используете приватный канал, бота нужно пригласить в
                  него вручную:
                </strong>
              </p>

              <ol>
                <li>
                  В приватном канале введите{" "}
                  <code>@Databasus Notifications</code> (или имя, которое вы
                  дали приложению)
                </li>
                <li>
                  Нажмите на имя бота, когда оно появится, и выберите{" "}
                  <strong>&quot;Add to Channel&quot;</strong> или{" "}
                  <strong>&quot;Invite to Channel&quot;</strong>
                </li>
              </ol>

              <p>
                В <strong>публичные каналы</strong> бот подключится
                автоматически при отправке первого сообщения (благодаря праву{" "}
                <code>channels:join</code>), поэтому для них этот шаг не нужен.
              </p>

              <h2 id="configure-databasus">Настройте Databasus</h2>

              <h3 id="add-slack-notifier">13. Добавьте Slack-уведомление</h3>

              <p>
                В Databasus откройте настройки уведомлений и добавьте новое
                Slack-уведомление:
              </p>

              <ul>
                <li>
                  <strong>Bot Token:</strong> вставьте скопированный Bot User
                  OAuth Token (начинается с <code>xoxb-</code>)
                </li>
                <li>
                  <strong>Target Channel ID:</strong> вставьте скопированный
                  Channel ID (начинается с <code>C</code>, <code>G</code>,{" "}
                  <code>D</code> или <code>U</code>)
                </li>
              </ul>

              <img
                src="/images/notifier-slack/image-5.png"
                alt="Добавление Slack-уведомления"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h3 id="test-notification">14. Проверьте уведомление</h3>

              <p>
                После настройки отправьте тестовое уведомление и убедитесь, что
                все работает. Тестовое сообщение должно прийти в выбранный канал
                Slack.
              </p>

              <p>
                Готово! Теперь уведомления о бекапах PostgreSQL из Databasus
                будут приходить в ваш Slack.
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
