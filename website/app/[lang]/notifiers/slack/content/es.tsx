import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Cómo configurar notificaciones de Slack en Databasus | Databasus",
  description:
    "Guía paso a paso para configurar notificaciones de Slack sobre las copias de seguridad de PostgreSQL con Databasus. Aprenda a crear una app de bot de Slack y a configurar las notificaciones.",
  keywords: [
    "Databasus",
    "notificaciones de Slack",
    "copia de seguridad de PostgreSQL",
    "token de bot de Slack",
    "Slack API",
    "alertas de copia de seguridad",
    "notificaciones de base de datos",
  ],
  openGraph: {
    title: "Cómo configurar notificaciones de Slack en Databasus | Databasus",
    description:
      "Guía paso a paso para configurar notificaciones de Slack sobre las copias de seguridad de PostgreSQL con Databasus. Aprenda a crear una app de bot de Slack y a configurar las notificaciones.",
    type: "article",
    url: getLocalizedUrl("es", "notifiers/slack"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title: "Cómo configurar notificaciones de Slack en Databasus | Databasus",
    description:
      "Guía paso a paso para configurar notificaciones de Slack sobre las copias de seguridad de PostgreSQL con Databasus. Aprenda a crear una app de bot de Slack y a configurar las notificaciones.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "notifiers/slack"),
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
            name: "Cómo configurar notificaciones de Slack en Databasus",
            description:
              "Guía paso a paso para configurar notificaciones de Slack sobre las copias de seguridad de PostgreSQL con Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Vaya a la Slack API",
                text: "Abra https://api.slack.com/apps e inicie sesión en su espacio de trabajo de Slack.",
              },
              {
                "@type": "HowToStep",
                name: "Cree una nueva app",
                text: "Haga clic en el botón 'Create New App' y elija 'From scratch'.",
              },
              {
                "@type": "HowToStep",
                name: "Configure los permisos del bot",
                text: "Vaya a OAuth & Permissions y añada los scopes requeridos en Bot Token Scopes: chat:write, channels:join, im:write y groups:write.",
              },
              {
                "@type": "HowToStep",
                name: "Instale la app en el espacio de trabajo",
                text: "Instale la app en su espacio de trabajo y autorícela.",
              },
              {
                "@type": "HowToStep",
                name: "Copie el token del bot",
                text: "Copie el Bot User OAuth Token que empieza por 'xoxb-'.",
              },
              {
                "@type": "HowToStep",
                name: "Obtenga el ID del canal",
                text: "Abra el canal de destino y obtenga el Channel ID desde los detalles del canal.",
              },
              {
                "@type": "HowToStep",
                name: "Añada el bot a un canal privado",
                text: "Si usa un canal privado, invite al bot al canal mencionándolo.",
              },
              {
                "@type": "HowToStep",
                name: "Configure Databasus",
                text: "En Databasus, añada el Bot Token y el Channel ID en la configuración del notificador de Slack.",
              },
              {
                "@type": "HowToStep",
                name: "Pruebe la notificación",
                text: "Pruebe la notificación para comprobar que funciona correctamente.",
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
              <h1 id="slack-notifications">Notificaciones de Slack</h1>

              <p className="text-lg text-gray-400">
                Configure Slack para recibir al instante notificaciones sobre el
                estado de sus copias de seguridad de PostgreSQL. Reciba alertas
                de respaldos exitosos, fallos y advertencias directamente en sus
                canales de Slack.
              </p>

              <h2 id="create-slack-app">Cree una app de Slack</h2>

              <h3 id="go-to-slack-api">1. Vaya a la Slack API</h3>

              <p>
                Abra{" "}
                <a
                  href="https://api.slack.com/apps"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  https://api.slack.com/apps
                </a>{" "}
                e inicie sesión en su espacio de trabajo de Slack.
              </p>

              <h3 id="create-new-app">2. Cree una nueva app</h3>

              <p>
                Haga clic en el botón{" "}
                <strong>&quot;Create New App&quot;</strong>.
              </p>

              <h3 id="choose-from-scratch">
                3. Elija &quot;From scratch&quot;
              </h3>

              <p>
                Seleccione la opción <strong>&quot;From scratch&quot;</strong>{" "}
                cuando se le pregunte.
              </p>

              <h3 id="name-your-app">4. Asigne un nombre a la app</h3>

              <p>
                Escriba un nombre para la app (por ejemplo, &quot;Databasus
                Notifications&quot;) y seleccione el espacio de trabajo donde
                quiere instalarla. Haga clic en{" "}
                <strong>&quot;Create App&quot;</strong>.
              </p>

              <h2 id="configure-bot-permissions">
                Configure los permisos del bot
              </h2>

              <h3 id="navigate-to-oauth">5. Vaya a OAuth &amp; Permissions</h3>

              <p>
                En la barra lateral izquierda, haga clic en{" "}
                <strong>&quot;OAuth &amp; Permissions&quot;</strong>.
              </p>

              <img
                src="/images/notifier-slack/image-1.png"
                alt="Ir a OAuth &amp; Permissions"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h3 id="add-bot-scopes">
                6. Añada los Bot Token Scopes (obligatorio)
              </h3>

              <p>
                Baje hasta la sección <strong>&quot;Scopes&quot;</strong> y, en{" "}
                <strong>&quot;Bot Token Scopes&quot;</strong>, haga clic en{" "}
                <strong>&quot;Add an OAuth Scope&quot;</strong>.
              </p>

              <p>Añada todos estos scopes obligatorios:</p>

              <ul>
                <li>
                  <code>chat:write</code> - para enviar mensajes a los canales
                </li>
                <li>
                  <code>channels:join</code> - para que el bot pueda unirse
                  automáticamente a los canales públicos
                </li>
                <li>
                  <code>im:write</code> - para enviar mensajes directos a los
                  usuarios
                </li>
                <li>
                  <code>groups:write</code> - para enviar mensajes a canales
                  privados
                </li>
                <li>
                  <code>channels:history</code> - para leer el historial del
                  canal
                </li>
              </ul>

              <img
                src="/images/notifier-slack/image-2.png"
                alt="Añadir Bot Token Scopes"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h2 id="install-app">Instale la app en el espacio de trabajo</h2>

              <h3 id="install-to-workspace">
                7. Instale en el espacio de trabajo
              </h3>

              <p>
                Suba al inicio de la página{" "}
                <strong>&quot;OAuth &amp; Permissions&quot;</strong> y haga clic
                en <strong>&quot;Install to Workspace&quot;</strong>.
              </p>

              <img
                src="/images/notifier-slack/image-3.png"
                alt="Instalar en el espacio de trabajo"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h3 id="authorize-app">8. Autorice la app</h3>

              <p>
                Revise los permisos y haga clic en{" "}
                <strong>&quot;Allow&quot;</strong> para autorizar la app.
              </p>

              <h3 id="copy-bot-token">9. Copie el Bot User OAuth Token</h3>

              <p>
                Tras la instalación verá el{" "}
                <strong>&quot;Bot User OAuth Token&quot;</strong>. Empieza por{" "}
                <code>xoxb-</code>. Copie este token: lo necesitará para
                configurar Databasus.
              </p>

              <h2 id="get-channel-id">Obtenga el ID del canal</h2>

              <h3 id="open-channel">10. Abra el canal de destino</h3>

              <p>
                En su espacio de trabajo de Slack, abra el canal donde quiere
                recibir las notificaciones de respaldo.
              </p>

              <h3 id="get-channel-info">11. Obtenga el ID del canal</h3>

              <p>
                Haga clic en el nombre del canal en la parte superior y baje en
                los detalles del canal. Encontrará el{" "}
                <strong>Channel ID</strong> al final de la sección
                &quot;About&quot;. Empieza por <code>C</code> (canales públicos)
                o <code>G</code> (canales privados).
              </p>

              <p>Copie este Channel ID.</p>

              <img
                src="/images/notifier-slack/image-4.png"
                alt="Obtener el Channel ID"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[500px]"
                loading="lazy"
              />

              <h3 id="add-bot-to-channel">
                12. Añada el bot al canal (obligatorio para canales privados)
              </h3>

              <p>
                <strong>
                  Si usa un canal privado, debe invitar al bot al canal
                  manualmente:
                </strong>
              </p>

              <ol>
                <li>
                  En el canal privado, escriba{" "}
                  <code>@Databasus Notifications</code> (o el nombre que le haya
                  dado a su app)
                </li>
                <li>
                  Haga clic en el nombre del bot cuando aparezca y seleccione{" "}
                  <strong>&quot;Add to Channel&quot;</strong> o{" "}
                  <strong>&quot;Invite to Channel&quot;</strong>
                </li>
              </ol>

              <p>
                En los <strong>canales públicos</strong> el bot se une
                automáticamente al enviar el primer mensaje (gracias al permiso{" "}
                <code>channels:join</code>), así que este paso no es necesario.
              </p>

              <h2 id="configure-databasus">Configure Databasus</h2>

              <h3 id="add-slack-notifier">13. Añada el notificador de Slack</h3>

              <p>
                En Databasus, vaya a los ajustes de notificadores y añada un
                nuevo notificador de Slack:
              </p>

              <ul>
                <li>
                  <strong>Bot Token:</strong> pegue el Bot User OAuth Token que
                  copió (empieza por <code>xoxb-</code>)
                </li>
                <li>
                  <strong>Target Channel ID:</strong> pegue el Channel ID que
                  copió (empieza por <code>C</code>, <code>G</code>,{" "}
                  <code>D</code> o <code>U</code>)
                </li>
              </ul>

              <img
                src="/images/notifier-slack/image-5.png"
                alt="Añadir el notificador de Slack"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[700px]"
                loading="lazy"
              />

              <h3 id="test-notification">14. Pruebe la notificación</h3>

              <p>
                Después de configurar el notificador, pruébelo para comprobar
                que funciona correctamente. Debería recibir un mensaje de prueba
                en el canal de Slack seleccionado.
              </p>

              <p>
                ¡Eso es todo! Su espacio de trabajo de Slack ya está configurado
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
