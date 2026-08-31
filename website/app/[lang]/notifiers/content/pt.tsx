import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Notificadores - Documentação do Databasus",
  description:
    "Lista de canais de notificação suportados para alertas de backup do Databasus, incluindo Slack, Discord, Telegram, Microsoft Teams, Email e Webhooks.",
  keywords: [
    "notificadores Databasus",
    "notificações de backup",
    "notificações Slack",
    "alertas Discord",
    "notificações Telegram",
    "notificações Teams",
    "alertas por email",
    "notificações webhook",
  ],
  openGraph: {
    title: "Notificadores - Documentação do Databasus",
    description:
      "Lista de canais de notificação suportados para alertas de backup do Databasus, incluindo Slack, Discord, Telegram, Microsoft Teams, Email e Webhooks.",
    type: "article",
    url: getLocalizedUrl("pt", "notifiers"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title: "Notificadores - Documentação do Databasus",
    description:
      "Lista de canais de notificação suportados para alertas de backup do Databasus, incluindo Slack, Discord, Telegram, Microsoft Teams, Email e Webhooks.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "notifiers"),
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
            headline: "Notificadores - Documentação do Databasus",
            description:
              "Lista de canais de notificação suportados para alertas de backup do Databasus, incluindo Slack, Discord, Telegram, Microsoft Teams, Email e Webhooks.",
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

      <DocsNavbarComponent lang="pt" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="pt" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="notifiers">Notificadores</h1>

              <p className="text-lg text-gray-400">
                O Databasus suporta vários canais de notificação para manter
                você informado sobre o status do seu backup PostgreSQL. Receba
                alertas instantâneos quando os backups terminam com sucesso,
                falham ou encontram problemas.
              </p>

              <h2 id="supported-notifiers">Notificadores suportados</h2>

              <ul>
                <li>
                  <a
                    href="/pt/notifiers/slack"
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Slack
                  </a>{" "}
                  - Envie notificações para canais do Slack via webhooks
                </li>
                <li>
                  <strong>Discord</strong> - Publique alertas de backup em
                  canais do Discord
                </li>
                <li>
                  <strong>Telegram</strong> - Receba notificações através de
                  bots do Telegram
                </li>
                <li>
                  <a
                    href="/pt/notifiers/teams"
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Microsoft Teams
                  </a>{" "}
                  - Notifique a sua equipe através de canais do Microsoft Teams
                </li>
                <li>
                  <strong>Email</strong> - Envie notificações por email sobre
                  eventos de backup
                </li>
                <li>
                  <strong>Webhook</strong> - Integração personalizada por
                  webhook com qualquer serviço
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
