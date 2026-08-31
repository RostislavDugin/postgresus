import type { Metadata } from "next";
import {
  OG_LOCALES,
  getLanguageAlternates,
  getLocalizedUrl,
} from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import Image from "next/image";

export const metadata: Metadata = {
  title:
    "Cómo configurar notificaciones de Microsoft Teams en Databasus | Databasus",
  description:
    "Guía paso a paso para configurar notificaciones de Microsoft Teams sobre las copias de seguridad de PostgreSQL con Databasus. Aprenda a crear un webhook de Teams y a configurar las notificaciones.",
  keywords: [
    "Databasus",
    "notificaciones de Microsoft Teams",
    "copia de seguridad de PostgreSQL",
    "webhook de Teams",
    "alertas de copia de seguridad",
    "notificaciones de base de datos",
  ],
  openGraph: {
    title:
      "Cómo configurar notificaciones de Microsoft Teams en Databasus | Databasus",
    description:
      "Guía paso a paso para configurar notificaciones de Microsoft Teams sobre las copias de seguridad de PostgreSQL con Databasus. Aprenda a crear un webhook de Teams y a configurar las notificaciones.",
    type: "article",
    url: getLocalizedUrl("es", "notifiers/teams"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title:
      "Cómo configurar notificaciones de Microsoft Teams en Databasus | Databasus",
    description:
      "Guía paso a paso para configurar notificaciones de Microsoft Teams sobre las copias de seguridad de PostgreSQL con Databasus. Aprenda a crear un webhook de Teams y a configurar las notificaciones.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "notifiers/teams"),
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
            name: "Cómo configurar notificaciones de Microsoft Teams en Databasus",
            description:
              "Guía paso a paso para configurar notificaciones de Microsoft Teams sobre las copias de seguridad de PostgreSQL con Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Abra el canal de Teams",
                text: "Vaya al canal de Microsoft Teams donde quiere recibir las notificaciones.",
              },
              {
                "@type": "HowToStep",
                name: "Acceda a los flujos de trabajo",
                text: "Abra la función Workflows en su canal de Teams.",
              },
              {
                "@type": "HowToStep",
                name: "Cree un nuevo flujo de trabajo",
                text: "Cree un nuevo flujo de trabajo para webhooks entrantes.",
              },
              {
                "@type": "HowToStep",
                name: "Seleccione la plantilla de webhook",
                text: "Elija la plantilla de webhook entrante entre las opciones disponibles.",
              },
              {
                "@type": "HowToStep",
                name: "Configure el webhook",
                text: "Defina el nombre del webhook y el canal.",
              },
              {
                "@type": "HowToStep",
                name: "Copie la URL del webhook",
                text: "Copie la URL del webhook generada por Teams.",
              },
              {
                "@type": "HowToStep",
                name: "Configure Databasus",
                text: "Pegue la URL del webhook en la configuración del notificador de Databasus.",
              },
            ],
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
              <h1 id="teams-notifications">
                Notificaciones de Microsoft Teams
              </h1>

              <p className="text-lg text-gray-400">
                Configure Microsoft Teams para recibir al instante
                notificaciones sobre el estado de sus copias de seguridad de
                PostgreSQL. Reciba alertas de respaldos exitosos, fallos y
                advertencias directamente en sus canales de Teams.
              </p>

              <h2 id="setup-teams-webhook">Configure el webhook de Teams</h2>

              <h3 id="open-teams-channel">1. Abra su canal de Teams</h3>

              <p>
                Vaya al canal de Microsoft Teams donde quiere recibir las
                notificaciones de respaldo. Haga clic en los tres puntos (
                <strong>•••</strong>) junto al nombre del canal.
              </p>

              <Image
                src="/images/notifier-teams/image-01.png"
                alt="Abrir el canal de Teams"
                width={800}
                height={500}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="access-workflows">2. Acceda a los flujos de trabajo</h3>

              <p>
                En el menú del canal, seleccione{" "}
                <strong>&quot;Workflows&quot;</strong> para abrir la integración
                con Power Automate.
              </p>

              <Image
                src="/images/notifier-teams/image-02.png"
                alt="Acceder a Workflows"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="create-new-workflow">3. Cree un nuevo flujo de trabajo</h3>

              <p>
                En el panel de Workflows, haga clic en{" "}
                <strong>&quot;Create&quot;</strong> o busque la plantilla{" "}
                <strong>
                  &quot;Post to a channel when a webhook request is
                  received&quot;
                </strong>
                .
              </p>

              <Image
                src="/images/notifier-teams/image-03.png"
                alt="Crear un nuevo flujo de trabajo"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="select-webhook-template">
                4. Seleccione la plantilla de webhook
              </h3>

              <p>
                Elija la plantilla{" "}
                <strong>
                  &quot;Post to a channel when a webhook request is
                  received&quot;
                </strong>{" "}
                entre las opciones disponibles.
              </p>

              <Image
                src="/images/notifier-teams/image-04.png"
                alt="Seleccionar la plantilla de webhook"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="configure-webhook">5. Configure el webhook</h3>

              <p>
                Configure el webhook indicando un nombre (por ejemplo,{" "}
                <strong>&quot;Databasus Backup Notifications&quot;</strong>) y
                confirme el canal donde se publicarán las notificaciones.
              </p>

              <Image
                src="/images/notifier-teams/image-05.png"
                alt="Configurar el webhook"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="copy-webhook-url">6. Copie la URL del webhook</h3>

              <p>
                Tras crear el flujo de trabajo, verá la{" "}
                <strong>HTTP POST URL</strong>. Copie esta URL: la necesitará
                para configurar Databasus.
              </p>

              <Image
                src="/images/notifier-teams/image-06.png"
                alt="Copiar la URL del webhook"
                width={500}
                height={500}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h2 id="configure-databasus">Configure Databasus</h2>

              <h3 id="add-teams-notifier">
                1. Añada el notificador de Teams
              </h3>

              <p>
                En Databasus, vaya a los ajustes de notificadores y añada un
                nuevo notificador de Microsoft Teams. Pegue la URL del webhook
                que copió de Teams.
              </p>

              <Image
                src="/images/notifier-teams/image-07.png"
                alt="Configurar Teams en Databasus"
                width={500}
                height={400}
                className="my-6 rounded-lg border border-gray-200"
                loading="lazy"
              />

              <h3 id="test-notification">2. Pruebe la notificación</h3>

              <p>
                Después de configurar el webhook, pruebe la notificación para
                comprobar que funciona correctamente. Debería recibir un mensaje
                de prueba en el canal de Teams seleccionado.
              </p>

              <p>
                ¡Eso es todo! Su canal de Microsoft Teams ya está configurado
                para recibir notificaciones de las copias de seguridad de
                PostgreSQL desde Databasus.
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href="/es/notifiers"
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Volver a notificadores
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
