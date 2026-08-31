import type { Metadata } from "next";
import {
  OG_LOCALES,
  getLanguageAlternates,
  getLocalizedUrl,
} from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Notificateurs - Documentation Databasus",
  description:
    "Liste des canaux de notification pris en charge pour les alertes de sauvegarde Databasus : Slack, Discord, Telegram, Microsoft Teams, Email et Webhooks.",
  keywords: [
    "notificateurs Databasus",
    "notifications de sauvegarde",
    "notifications Slack",
    "alertes Discord",
    "notifications Telegram",
    "notifications Teams",
    "alertes Email",
    "notifications Webhook",
  ],
  openGraph: {
    title: "Notificateurs - Documentation Databasus",
    description:
      "Liste des canaux de notification pris en charge pour les alertes de sauvegarde Databasus : Slack, Discord, Telegram, Microsoft Teams, Email et Webhooks.",
    type: "article",
    url: getLocalizedUrl("fr", "notifiers"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title: "Notificateurs - Documentation Databasus",
    description:
      "Liste des canaux de notification pris en charge pour les alertes de sauvegarde Databasus : Slack, Discord, Telegram, Microsoft Teams, Email et Webhooks.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "notifiers"),
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
            headline: "Notificateurs - Documentation Databasus",
            description:
              "Liste des canaux de notification pris en charge pour les alertes de sauvegarde Databasus : Slack, Discord, Telegram, Microsoft Teams, Email et Webhooks.",
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

      <DocsNavbarComponent lang="fr" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="fr" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="notifiers">Notificateurs</h1>

              <p className="text-lg text-gray-400">
                Databasus prend en charge plusieurs canaux de notification pour
                vous tenir informé du statut de vos sauvegardes PostgreSQL.
                Recevez des alertes instantanées lorsque les backups
                réussissent, échouent ou rencontrent des problèmes.
              </p>

              <h2 id="supported-notifiers">Notificateurs pris en charge</h2>

              <ul>
                <li>
                  <a
                    href="/fr/notifiers/slack"
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Slack
                  </a>{" "}
                  - Envoyez des notifications vers des canaux Slack via webhooks
                </li>
                <li>
                  <strong>Discord</strong> - Publiez des alertes de sauvegarde
                  dans des canaux Discord
                </li>
                <li>
                  <strong>Telegram</strong> - Recevez des notifications via des
                  bots Telegram
                </li>
                <li>
                  <a
                    href="/fr/notifiers/teams"
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Microsoft Teams
                  </a>{" "}
                  - Notifiez votre équipe via des canaux Microsoft Teams
                </li>
                <li>
                  <strong>Email</strong> - Envoyez des notifications par e-mail
                  pour les événements de sauvegarde
                </li>
                <li>
                  <strong>Webhook</strong> - Intégration webhook personnalisée
                  pour n&apos;importe quel service
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
