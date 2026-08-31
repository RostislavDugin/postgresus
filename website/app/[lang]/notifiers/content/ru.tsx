import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Уведомления - документация Databasus",
  description:
    "Список поддерживаемых каналов уведомлений о бекапах в Databasus: Slack, Discord, Telegram, Microsoft Teams, Email и вебхуки.",
  keywords: [
    "уведомления Databasus",
    "уведомления о бекапах",
    "уведомления в Slack",
    "оповещения в Discord",
    "уведомления в Telegram",
    "уведомления в Teams",
    "оповещения по Email",
    "уведомления через вебхуки",
  ],
  openGraph: {
    title: "Уведомления - документация Databasus",
    description:
      "Список поддерживаемых каналов уведомлений о бекапах в Databasus: Slack, Discord, Telegram, Microsoft Teams, Email и вебхуки.",
    type: "article",
    url: getLocalizedUrl("ru", "notifiers"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title: "Уведомления - документация Databasus",
    description:
      "Список поддерживаемых каналов уведомлений о бекапах в Databasus: Slack, Discord, Telegram, Microsoft Teams, Email и вебхуки.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "notifiers"),
    languages: getLanguageAlternates("notifiers"),
  },
  robots: "index, follow",
};

export default function NotifiersPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Уведомления - документация Databasus",
            description:
              "Список поддерживаемых каналов уведомлений о бекапах в Databasus: Slack, Discord, Telegram, Microsoft Teams, Email и вебхуки.",
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
              <h1 id="notifiers">Уведомления</h1>

              <p className="text-lg text-gray-400">
                Databasus поддерживает несколько каналов уведомлений, чтобы вы
                всегда знали, как прошло резервное копирование PostgreSQL.
                Оповещения об успешных бекапах, сбоях и проблемах приходят
                мгновенно.
              </p>

              <h2 id="supported-notifiers">Поддерживаемые каналы</h2>

              <ul>
                <li>
                  <a
                    href="/ru/notifiers/slack"
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Slack
                  </a>{" "}
                  — отправка уведомлений в каналы Slack через вебхуки
                </li>
                <li>
                  <strong>Discord</strong> — публикация оповещений о бекапах в
                  каналах Discord
                </li>
                <li>
                  <strong>Telegram</strong> — получение уведомлений через
                  Telegram-ботов
                </li>
                <li>
                  <a
                    href="/ru/notifiers/teams"
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Microsoft Teams
                  </a>{" "}
                  — уведомления вашей команде в каналах Microsoft Teams
                </li>
                <li>
                  <strong>Email</strong> — отправка писем о событиях резервного
                  копирования
                </li>
                <li>
                  <strong>Webhook</strong> — собственная интеграция через вебхук
                  с любым сервисом
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
