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
  title: "Notificadores - Documentación de Databasus",
  description:
    "Lista de canales de notificación compatibles con las alertas de copia de seguridad de Databasus: Slack, Discord, Telegram, Microsoft Teams, Email y Webhooks.",
  keywords: [
    "notificadores Databasus",
    "notificaciones de copia de seguridad",
    "notificaciones de Slack",
    "alertas de Discord",
    "notificaciones de Telegram",
    "notificaciones de Teams",
    "alertas por Email",
    "notificaciones por Webhook",
  ],
  openGraph: {
    title: "Notificadores - Documentación de Databasus",
    description:
      "Lista de canales de notificación compatibles con las alertas de copia de seguridad de Databasus: Slack, Discord, Telegram, Microsoft Teams, Email y Webhooks.",
    type: "article",
    url: getLocalizedUrl("es", "notifiers"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title: "Notificadores - Documentación de Databasus",
    description:
      "Lista de canales de notificación compatibles con las alertas de copia de seguridad de Databasus: Slack, Discord, Telegram, Microsoft Teams, Email y Webhooks.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "notifiers"),
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
            headline: "Notificadores - Documentación de Databasus",
            description:
              "Lista de canales de notificación compatibles con las alertas de copia de seguridad de Databasus: Slack, Discord, Telegram, Microsoft Teams, Email y Webhooks.",
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

      <DocsNavbarComponent lang="es" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="es" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="notifiers">Notificadores</h1>

              <p className="text-lg text-gray-400">
                Databasus admite varios canales de notificación para mantenerle
                informado del estado de sus copias de seguridad de PostgreSQL.
                Reciba alertas al instante cuando un respaldo termina con
                éxito, falla o encuentra problemas.
              </p>

              <h2 id="supported-notifiers">Notificadores compatibles</h2>

              <ul>
                <li>
                  <a
                    href="/es/notifiers/slack"
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Slack
                  </a>{" "}
                  - Envíe notificaciones a canales de Slack mediante webhooks
                </li>
                <li>
                  <strong>Discord</strong> - Publique alertas de respaldo en
                  canales de Discord
                </li>
                <li>
                  <strong>Telegram</strong> - Reciba notificaciones a través de
                  bots de Telegram
                </li>
                <li>
                  <a
                    href="/es/notifiers/teams"
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Microsoft Teams
                  </a>{" "}
                  - Avise a su equipo por canales de Microsoft Teams
                </li>
                <li>
                  <strong>Email</strong> - Envíe notificaciones por correo sobre
                  los eventos de respaldo
                </li>
                <li>
                  <strong>Webhook</strong> - Integración por webhook
                  personalizado para cualquier servicio
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
